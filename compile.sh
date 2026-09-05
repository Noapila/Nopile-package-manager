#!/bin/bash

# Options par défaut
VERBOSE=0
YES=0

log() {
    if [ "$VERBOSE" -eq 1 ]; then
        echo "$1"
    fi
}

# Traiter les arguments avec un case (sans bloquer si aucune option)
for arg in "$@"; do
    case $arg in
        -v|--verbose)
            VERBOSE=1
            ;;
        -y|--yes)
            YES=1
            ;;
        -h|--help)
            echo "Usage: $0 [-v|--verbose] [-y|--yes]"
            exit 0
            ;;
        *)
            echo "Unknown option: $arg (ignored)"
            exit 1
            ;;
    esac
done

if [ $YES -eq 0 ]; then
    echo "=========================================="
    echo "  Hi, I'm Nopile-mini"
    echo "  I'm here to compile Nopile for you !"
    echo "=========================================="
    echo ""
    read -p "Do you want to continue? [y/N] " answer

    if [[ ! "$answer" =~ ^[Yy]$ ]]; then
        echo "aborting..."
        exit 0
    fi
fi

echo "==> [1/4] Extracting Go compiler"
log "nopile-mini: ./static/tar -x -I './static/zstd -d' -f go.tar.zst"

if ! ./static/tar -x -I './static/zstd -d' -f go.tar.zst; then
    echo "Error: failed to decompress go.tar.zst"
    echo "aborting..."
    exit 1
fi

echo "==> [2/4] Compiling Nopile"
log "nopile-mini: CGO_ENABLED=0 ./go/bin/go build -ldflags="-s -w" -o nopile"
CGO_ENABLED=0 ./go/bin/go build -ldflags="-s -w" -o nopile || exit 1

echo "==> [3/4] Creating install.sh"
cat > install.sh << 'EOF'
#!/bin/bash
echo "==> Installing Nopile"
mkdir -p /var/lib/nopile || exit 1
mkdir -p /var/tmp/nopile || exit 1
mkdir -p /var/cache/nopile || exit 1
cp ./nopile /usr/bin/nopile || exit 1
echo "Nopile installed Succesfully !"
EOF
chmod +x install.sh

echo "==> [4/4] Cleaning files"
log "nopile-mini: rm -rf $HOME/.cache/go-build/"
rm -rf $HOME/.cache/go-build/

echo 'Nopile compiled succesfully run install.sh to install it !'

