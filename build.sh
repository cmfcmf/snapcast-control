#!/bin/bash
set -e

echo "Building frontend..."
cd frontend-react
yarn install
yarn build
cd ..

echo "Building Go binary..."
go build -o snapcast-control

echo "Build complete! Run with: ./snapcast-control"
