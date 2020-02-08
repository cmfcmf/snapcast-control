#/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -ev

cd "$DIR/frontend-react"
yarn install
yarn build
cd "$DIR"

rsync -rtv -e "ssh -l pi" --filter=':- .gitignore' --exclude=".git" "$DIR/" 192.168.0.100:/home/pi/snapcast-control
rsync -rtv -e "ssh -l pi" "$DIR/frontend-react/build/" 192.168.0.100:/home/pi/snapcast-control/frontend-react/build