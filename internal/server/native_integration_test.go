package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeInstallerPreservesFilesAndUsesNativeProfile(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	for _, cli := range []string{"grok", "hermes"} {
		t.Run(cli, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("HOME", root)
			t.Setenv("GROK_HOME", "")
			t.Setenv("HERMES_HOME", "")
			data := filepath.Join(root, "data")
			if err := writeNativeAssets(data); err != nil {
				t.Fatal(err)
			}
			native := filepath.Join(root, "fake-hermes")
			script := `#!/bin/sh
case "$*" in
 '--profile selected config path') printf '%s\n' "$HOME/vendor-selected/config.yaml" ;;
 '--profile selected plugins enable picode-native --no-allow-tool-override') printf '%s\n' "$*" > "$HOME/enabled" ;;
 *) exit 2 ;;
esac
`
			if err := os.WriteFile(native, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			base := filepath.Join(root, ".grok")
			target := "hooks/picode-native.json"
			if cli == "hermes" {
				base = filepath.Join(root, "vendor-selected")
				target = "plugins/picode-native/__init__.py"
			}
			os.MkdirAll(base, 0700)
			sentinel := filepath.Join(base, "config.yaml")
			os.WriteFile(sentinel, []byte("user: preserved\n"), 0600)
			run := func() ([]byte, error) {
				return exec.Command("python3", filepath.Join(nativeAssetsDir(data), "install.py"), cli, nativeAssetsDir(data), native, "chat", "--profile", "selected").CombinedOutput()
			}
			if out, err := run(); err != nil {
				t.Fatalf("install: %v %s", err, out)
			}
			if out, err := run(); err != nil {
				t.Fatalf("idempotent install: %v %s", err, out)
			}
			if raw, _ := os.ReadFile(sentinel); string(raw) != "user: preserved\n" {
				t.Fatal("native settings overwritten")
			}
			if cli == "hermes" {
				if _, err := os.Stat(filepath.Join(root, ".hermes", "plugins")); !os.IsNotExist(err) {
					t.Fatal("installed into wrong profile")
				}
			}
			path := filepath.Join(base, target)
			os.WriteFile(path, []byte("owner edit"), 0600)
			if out, err := run(); err == nil || !strings.Contains(string(out), "was changed") {
				t.Fatalf("modified integration clobbered: %v %s", err, out)
			}
			if raw, _ := os.ReadFile(path); string(raw) != "owner edit" {
				t.Fatal("lost owner edit")
			}
			os.Remove(path)
			os.Symlink(sentinel, path)
			if out, err := run(); err == nil || !strings.Contains(string(out), "symlink") {
				t.Fatalf("followed integration symlink: %v %s", err, out)
			}
		})
	}
}

func TestNativeHermesTurnContextSurvivesBackgroundReview(t *testing.T) {
	dir := t.TempDir()
	if err := writeNativeAssets(dir); err != nil {
		t.Fatal(err)
	}
	script := `import importlib.util,os,json,sys,subprocess,threading
from unittest.mock import patch
spec=importlib.util.spec_from_file_location("plugin",sys.argv[1]);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
os.environ.update(PICODE_TERM_ID="fixture",PICODE_NATIVE_HOOK="hook",PICODE_MESSAGES_BIN="/qa/picode",HERMES_SESSION_ID="helper")
class Context:
 def __init__(self):self.hooks={}
 def register_hook(self,name,callback):self.hooks[name]=callback
c=Context();m.register(c);observations=[]
def record(args,**kw):observations.append((json.loads(args[-1]),int(kw['env']['PICODE_NATIVE_SESSION_SEQ'])))
root=dict(session_id="native",turn_id="root-turn",platform="cli",parent_session_id="")
command="/qa/picode messages send --to peer_1 --request-id test --body 'quoted # ; body'"
def tool(**kw):return c.hooks['pre_tool_call'](tool_name='terminal',args={'command':command,'timeout':17},**kw)
with patch.object(m.subprocess,'run',side_effect=record):
 c.hooks['pre_llm_call'](**root)
 assert observations[-1][0]=={'state':'working','session_id':'native'}
 got=tool(**root);assert got['args']=={'command':'HERMES_SESSION_ID=native '+command,'timeout':17}
 assert os.environ['HERMES_SESSION_ID']=='helper'
 for kw in [dict(root,parent_session_id='native'),dict(root,platform='gateway'),dict(root,turn_id=''),dict(root,session_id='')]:
  before=len(observations);c.hooks['pre_llm_call'](**kw);assert len(observations)==before
 assert tool(**root)==got
 helper=dict(root,turn_id='helper-turn')
 assert tool(**helper) is None
 before=len(observations);c.hooks['on_session_end'](**helper);assert len(observations)==before
 c.hooks['pre_approval_request'](**root);assert observations[-1][0]['state']=='needs-you'
 c.hooks['post_approval_response'](**root);assert observations[-1][0]['state']=='working'
 for text in ['echo unrelated','HERMES_SESSION_ID=foreign '+command,command+'; /qa/picode messages read',command+'\n/qa/picode messages read','picode messages read # first\npicode messages ack msg_1','\n'+command,command+'\n']:
  assert c.hooks['pre_tool_call'](tool_name='terminal',args={'command':text},**root) is None
 c.hooks['on_session_end'](**root);assert observations[-1][0]['state']=='idle'
 newer=dict(root,session_id='next',turn_id='next-turn')
 c.hooks['pre_llm_call'](**newer);assert tool(**root) is None
 before=len(observations);c.hooks['on_session_end'](**root);assert len(observations)==before
 c.hooks['on_session_reset'](session_id='reset',platform='cli',reason='new_session')
 assert tool(**newer) is None and observations[-1][0]=={'state':'idle','session_id':'reset'}
 assert [seq for _,seq in observations]==sorted(set(seq for _,seq in observations))
 c.hooks['pre_llm_call'](**root)
entered=threading.Event();release=threading.Event()
def delayed(args,**kw):
 if json.loads(args[-1])['state']=='idle':
  entered.set();assert release.wait(3)
 record(args,**kw)
with patch.object(m.subprocess,'run',side_effect=delayed):
 old=threading.Thread(target=lambda:c.hooks['on_session_end'](**root));old.start()
 assert entered.wait(3)
 c.hooks['pre_llm_call'](**newer)
 release.set();old.join(3);assert not old.is_alive()
 assert observations[-1][0]['state']=='idle'
 assert max(observations,key=lambda o:o[1])[0]=={'state':'working','session_id':'next'}
`
	cmd := exec.Command("python3", "-c", script, filepath.Join(nativeAssetsDir(dir), "hermes.py"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Hermes context: %v %s", err, out)
	}
}
