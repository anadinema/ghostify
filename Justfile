set shell := ["bash", "-c"]

# Project Variables
app := "ghostify"
app_sidecar := "ghost-spot"

# Build Variables
features     := "rodio-backend,media-control,image,fzf"
rs_binary  := "spotify-player/target/release/spotify_player"
rs_ci_target := "target/bin"
rs_arm64_darwin := "aarch64-apple-darwin"
rs_x86_64_darwin := "x86_64-apple-darwin"

# Default task: show available commands
default:
    @just --list

# Initialize or update the Rust submodule to the pinned tag
update:
    @echo "Updating submodules..."
    git submodule update --init --recursive

# Build sidecar CLI locally
build-go:
    go build -o dist/bin/{{app_sidecar}} ./cli

# Build spotify-player from submodule for the current arch only (faster for local dev)
build-rs:
    @echo "Building 🎧 spotify-player with features: {{features}}..."
    cargo build --release --manifest-path spotify-player/Cargo.toml --features {{features}}
    @mkdir -p dist/bin
    cp {{rs_binary}} dist/bin/{{app}}

# Build everything locally
build: clean build-rs build-go

# Run tests
test:
    go test ./...

# Dry-run release — builds everything, generates formula, but does NOT push
snapshot:
    goreleaser release --snapshot --clean

# Tag and push to trigger the release workflow in CI
# Usage: just release v0.1.0
release v:
    @echo "Tagging {{v}} and pushing..."
    git tag {{v}}
    git push origin {{v}}

# Remove dist/
clean:
    rm -rf dist/

create-ci-rs-target:
    @echo "📦 Setting up CI target directories for multi-arch builds..."
    @rm -rf target/bin/{{app}}_darwin_*
    @mkdir -p target/bin/{{app}}_darwin_amd64
    @mkdir -p target/bin/{{app}}_darwin_arm64

build-multiarch-rs: clean create-ci-rs-target
    @echo "Repackaging 🎧 spotify-player as 👻 ghostify for multiple architectures with features: {{features}}..."
    cargo build --release --manifest-path spotify-player/Cargo.toml --features {{features}} --target x86_64-apple-darwin
    cargo build --release --manifest-path spotify-player/Cargo.toml --features {{features}} --target aarch64-apple-darwin
    @cp spotify-player/target/{{rs_x86_64_darwin}}/release/spotify_player target/bin/{{app}}_darwin_amd64/{{app}}
    @cp spotify-player/target/{{rs_arm64_darwin}}/release/spotify_player target/bin/{{app}}_darwin_arm64/{{app}}
    @echo "🚀 Multi-arch builds complete. Binaries located in target/bin"
    @ls -lh target/bin

# Copy the rust build artifacts into the dist directory for archiving
prepare-archive:
    @echo "📦 Preparing archive for release..."
    @cp -r target/bin dist/
    @ls -lh dist
    @ls -lh dist/bin
    @echo "👻 {{app}} ready to be packaged and released!"
