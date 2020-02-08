# Snapcast Control Web Interface

A web interface for [Snapcast](https://github.com/badaix/snapcast).
It allows you select which stream is played on which client.
It also has support to select local radio and files from Mopidy instances.

## Installation

```bash
sudo apt install python3 python3-pip
sudo pip3 install -r requirements.txt

cd frontend-react
yarn install && yarn build
```

Add an entry to crontab to start snapcast-control after booting:

```bash
sudo crontab -e # sudo is only needed for ports < 1000
```

```crontab
@reboot sleep 10 && /absolute/path/to/snapcast-control/server.py --port 80
```

## Development

Server:

```
sudo apt install python3 python3-venv
python3 -m venv .venv
source .venv/bin/activate
python server.py --debug --port 8080
```

Client:

```
cd frontend
yarn start
```
