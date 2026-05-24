#!/bin/bash
# Build a macOS .pkg installer for AgentX
# Usage: ./build-pkg.sh <binary-path> <version> <arch>
# Example: ./build-pkg.sh ./agentx 0.3.0 arm64
set -euo pipefail

BINARY="$1"
VERSION="$2"
ARCH="${3:-arm64}"  # arm64 or x86_64

IDENTIFIER="network.agentx.agentx"
INSTALL_LOCATION="/usr/local/bin"
PKG_NAME="AgentX-${VERSION}-${ARCH}.pkg"

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

# Create payload
PAYLOAD_DIR="$WORK_DIR/payload"
mkdir -p "$PAYLOAD_DIR/$INSTALL_LOCATION"
cp "$BINARY" "$PAYLOAD_DIR/$INSTALL_LOCATION/agentx"
chmod 755 "$PAYLOAD_DIR/$INSTALL_LOCATION/agentx"

# Create post-install script that opens a terminal with onboard
SCRIPTS_DIR="$WORK_DIR/scripts"
mkdir -p "$SCRIPTS_DIR"
cat > "$SCRIPTS_DIR/postinstall" << 'POSTINSTALL'
#!/bin/bash
# Open Terminal and run onboard wizard after install
if [ -x /usr/local/bin/agentx ]; then
    osascript -e '
        tell application "Terminal"
            activate
            do script "/usr/local/bin/agentx onboard"
        end tell
    ' 2>/dev/null || true
fi
exit 0
POSTINSTALL
chmod 755 "$SCRIPTS_DIR/postinstall"

# Build component package
COMPONENT_PKG="$WORK_DIR/agentx-component.pkg"
pkgbuild \
    --root "$PAYLOAD_DIR" \
    --identifier "$IDENTIFIER" \
    --version "$VERSION" \
    --install-location "/" \
    --scripts "$SCRIPTS_DIR" \
    "$COMPONENT_PKG"

# Create distribution XML for a nicer installer UI
DIST_XML="$WORK_DIR/distribution.xml"
cat > "$DIST_XML" << DIST
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>AgentX</title>
    <welcome language="en" mime-type="text/html"><![CDATA[
        <html><body style="font-family:-apple-system,sans-serif; padding:20px;">
        <h1 style="font-size:28px;">Welcome to AgentX</h1>
        <p style="font-size:15px; color:#555; line-height:1.6;">
            AgentX is an autonomous AI agent framework — the most lightweight agent runtime ever built.
        </p>
        <p style="font-size:14px; color:#777; line-height:1.6;">
            This installer will place the <code>agentx</code> binary in <code>/usr/local/bin</code>
            and launch the setup wizard when done.
        </p>
        <p style="font-size:13px; color:#999; margin-top:20px;">
            Version ${VERSION} &bull; agentx.network
        </p>
        </body></html>
    ]]></welcome>
    <options customize="never" require-scripts="false" hostArchitectures="arm64,x86_64"/>
    <choices-outline>
        <line choice="default"/>
    </choices-outline>
    <choice id="default" title="AgentX">
        <pkg-ref id="$IDENTIFIER"/>
    </choice>
    <pkg-ref id="$IDENTIFIER" version="$VERSION">#agentx-component.pkg</pkg-ref>
</installer-gui-script>
DIST

# Build product archive (final .pkg with UI)
productbuild \
    --distribution "$DIST_XML" \
    --package-path "$WORK_DIR" \
    "$PKG_NAME"

echo "Built: $PKG_NAME"
