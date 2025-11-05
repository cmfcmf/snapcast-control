#!/bin/bash
set -e

echo "Building snapcast-control for ARM (armhf/arm32)..."

# Ensure frontend is built
if [ ! -d "frontend-react/build" ]; then
    echo "Frontend build not found. Building frontend..."
    cd frontend-react
    yarn install
    yarn build
    cd ..
fi

# Build for ARM32 (armhf)
echo "Building for linux/arm (32-bit)..."
GOOS=linux GOARCH=arm GOARM=7 go build -o snapcast-control-armhf

echo "Build complete!"
echo "Binary: snapcast-control-armhf"
ls -lh snapcast-control-armhf
