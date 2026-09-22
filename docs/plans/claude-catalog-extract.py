# Extracts Claude Code's keymap catalog from the vendor docs page into
# claude-catalog-extracted.json (docs/plans/). Re-run after a claude release to
# re-check: python3 docs/plans/claude-catalog-extract.py > /tmp/cc.html.out
# (fetch https://code.claude.com/docs/en/keybindings to /tmp/cc-keybindings.html first).
import re, pathlib, html as H, sys
h = pathlib.Path("/tmp/cc-keybindings.html").read_text(errors="replace")
CTX = {"App actions": "Global", "History actions": None, "Chat actions": "Chat",
       "Autocomplete actions": "Autocomplete", "Confirmation actions": "Confirmation",
       "Permission actions": "Confirmation", "Transcript actions": "Transcript",
       "History search actions": "HistorySearch", "Task actions": "Task",
       "Theme actions": "ThemePicker", "Help actions": "Help", "Tabs actions": "Tabs",
       "Attachments actions": "Attachments", "Footer actions": "Footer",
       "Message selector actions": "MessageSelector", "Diff actions": "DiffDialog",
       "Diff panel actions": "DiffPanel", "Model picker actions": "ModelPicker",
       "Effort slider actions": "EffortSlider", "Select actions": "Select",
       "Plugin actions": "Plugin", "Agents actions": "Agents", "Scroll actions": "Scroll",
       "Settings actions": "Settings"}
def cells(tr):
    cs = re.findall(r"<t[dh][^>]*>(.*?)</t[dh]>", tr, re.S)
    return [re.sub(r"\s+", " ", H.unescape(re.sub(r"<[^>]+>", " ", c))).strip() for c in cs]
parts = re.split(r"(<h3[^>]*>.*?</h3>)", h, flags=re.S)
catalog, section = [], None
for p in parts:
    if p.startswith("<h3"):
        section = re.sub(r"<[^>]+>", "", p).replace("\u200b", "").strip()
        continue
    if section and section in CTX:
        for tr in re.findall(r"<tr[^>]*>(.*?)</tr>", p, re.S):
            c = cells(tr)
            if len(c) >= 2 and c[0] and not c[0].startswith("Action"):
                catalog.append({"ctx": CTX[section], "id": c[0],
                                "default": c[1] if len(c) > 1 else "",
                                "desc": c[2] if len(c) > 2 else ""})
out = sys.stdout if len(sys.argv) < 2 else open(sys.argv[1], "w")
json.dump(catalog, out, indent=1)
print(f"{len(catalog)} rows", file=sys.stderr)
