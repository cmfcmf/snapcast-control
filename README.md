# Snapcast Control Web Interface


## Installation

```bash
apt install python3 python3-venv
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

cd frontend
yarn install
yarn run ng build
```

## Development

Go to http://localhost:4200

### Server

```
source .venv/bin/activate
python server.py --debug --port 8080
```

### Client

```
cd frontend
yarn run ng serve
```
