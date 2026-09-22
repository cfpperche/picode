package install

import "fmt"

// NodeMajor is the Node.js major PiCode installs where it installs one (the
// Windows runtime stage, a shared-server member container): the version CI
// builds with. The CLIs a user adds from Agent CLIs need a current Node — Pi
// asks for >=22.19 (ADR-0179).
const NodeMajor = "22"

// NodeSourceScript adds the NodeSource apt repository for major and
// installs nodejs from it. It replicates the key, source, and pin lines of
// NodeSource's own setup script as our argv instead of piping their script
// (ADR-0093 refuses executing a vendor's curl|bash on the user's behalf),
// and ends with the install so one wsl call converges node end to end.
// Runs as root; major must be digits. No `$`: the architecture gates the
// run through a bare grep (anything else fails closed under set -e), and
// the source file omits Architectures — without it apt resolves the native
// arch, which the gate just proved supported.
func NodeSourceScript(major string) (string, error) {
	if major == "" {
		return "", fmt.Errorf("node major is empty")
	}
	for _, r := range major {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("node major is not a number: %q", major)
		}
	}
	return fmt.Sprintf(`set -e
dpkg --print-architecture | grep -qxE '(amd64|arm64)'
mkdir -p /usr/share/keyrings
rm -f /usr/share/keyrings/nodesource.gpg /etc/apt/sources.list.d/nodesource.list /etc/apt/sources.list.d/nodesource.sources
curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /usr/share/keyrings/nodesource.gpg
chmod 644 /usr/share/keyrings/nodesource.gpg
echo 'Types: deb' > /etc/apt/sources.list.d/nodesource.sources
echo 'URIs: https://deb.nodesource.com/node_%s.x' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Suites: nodistro' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Components: main' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Signed-By: /usr/share/keyrings/nodesource.gpg' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Package: nodejs' > /etc/apt/preferences.d/nodejs
echo 'Pin: origin deb.nodesource.com' >> /etc/apt/preferences.d/nodejs
echo 'Pin-Priority: 600' >> /etc/apt/preferences.d/nodejs
apt-get update
apt-get install -y nodejs
`, major), nil
}
