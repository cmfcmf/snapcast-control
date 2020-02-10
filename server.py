#!/usr/bin/python3
import json
import argparse
import sys
import time
import logging
import os
from typing import Optional, List, Dict

import tornado.ioloop
import tornado.gen
import tornado.web
from snapcast.control import Snapserver
from tornado.escape import to_unicode
from tornado.httpclient import AsyncHTTPClient
from tornado.platform.asyncio import AsyncIOMainLoop
import asyncio
from zeroconf import Zeroconf, ServiceListener, ServiceInfo
from serializer import Serializer

SNAPCAST_ZERO_NAME = '_snapcast-tcp._tcp.local.'

def noop(*args, **kws):
    return None

class ZeroListener(ServiceListener):
    def __init__(self, container: list, on_add_service = noop, on_remove_service = noop):
        self.container = container
        self.on_add_service = on_add_service
        self.on_remove_service = on_remove_service

    def add_service(self, zeroconf, type, name):
        info = zeroconf.get_service_info(type, name)
        logging.info("Service %s added, service info: %s" % (name, info))
        self.container.append(info)
        self.on_add_service(info)

    def update_service(self, zeroconf, type, name):
        info = zeroconf.get_service_info(type, name)
        logging.info("Service %s updated, service info: %s" % (name, info))

    def remove_service(self, zeroconf, type, name):
        logging.info("Service %s removed" % name)
        for i, info in enumerate(self.container):
            if info.name == name:
                self.container.pop(i)
                self.on_remove_service(info)
                break


class BaseHandler(tornado.web.RequestHandler):
    def initialize(self):
        super().initialize()
        self.serializer = Serializer()

    def set_default_headers(self, *args, **kwargs):
        self.set_header('Access-Control-Allow-Origin', '*')
        self.set_header('Content-Type', 'application/json')

    async def mopidy_rpc_request(self, server_name, method, params=None):
        body = json.dumps({
            "method": method,
            "jsonrpc": "2.0",
            "params": params if params is not None else {},
            "id": 1
        })

        headers = dict()
        headers['Content-Type'] = 'application/json'

        mopidy_server = self.get_mopidy_server_from_name(server_name)
        url = 'http://{}:{}/mopidy/rpc'.format(mopidy_server.server, mopidy_server.port)

        response = await http_client.fetch(url, method='POST', body=body, headers=headers)

        return json.loads(to_unicode(response.body))['result']

    @staticmethod
    def get_mopidy_server_from_name(name):
        return list(filter(lambda mopidy_server: mopidy_server.name == name, zero_mopidy_servers))[0]

    def write_json(self, data):
        self.write(json.dumps(self.serializer.serialize(data)))


class BrowseHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        uri = self.get_argument('uri', None)
        name = self.get_argument('name')

        items = yield self.mopidy_rpc_request(name, "core.library.browse", {'uri': uri})

        self.write_json(items)


class PlayHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        name = self.get_argument('name')
        uris = self.get_arguments('uri')
        yield self.mopidy_rpc_request(name, "core.tracklist.clear")
        tracks = yield self.mopidy_rpc_request(name, "core.tracklist.add", {'uris': uris})
        yield self.mopidy_rpc_request(name, "core.playback.play", {'tlid': tracks[0]['tlid']})

        self.write_json({})


class MopidyStopPlaybackHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        name = self.get_argument('name')
        yield self.mopidy_rpc_request(name, "core.tracklist.clear")
        yield self.mopidy_rpc_request(name, "core.playback.stop")
        self.write_json({})


class SnapServersHandler(BaseHandler):
    def get(self):
        self.write_json({name: { 'streams': server.streams, 'clients': server.clients } for name, server in snap_servers.items()})


class MopidyServersHandler(BaseHandler):
    def get(self):
        self.write_json(zero_mopidy_servers)


class ClientSettingsHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        server_name = self.get_argument('server_name')
        client_id = self.get_argument('id')
        action = self.get_argument('action')

        server = snap_servers[server_name]
        client = server.client(client_id)
        if action == 'mute':
            yield from client.set_muted(True)
        elif action == 'unmute':
            yield from client.set_muted(False)
        elif action == 'delete':
            yield from server.delete_client(client.identifier)
        elif action == 'set_latency':
            latency = int(self.get_argument('latency'))
            yield from client.set_latency(latency)
        elif action == 'set_stream':
            stream_id = self.get_argument('stream')
            yield from client.group.set_stream(stream_id)
        else:
            logging.error('Unknown action!')
            pass

        self.write_json({})


class StaticFileHandler(tornado.web.StaticFileHandler):
    def validate_absolute_path(self, root: str, absolute_path: str) -> Optional[str]:
        try:
            return super().validate_absolute_path(root, absolute_path)
        except tornado.web.HTTPError as e:
            if self.request.method == "GET" and e.status_code == 404:
                self.redirect("/")
                return None
            else:
                raise e


def make_app(debug):
    return tornado.web.Application([
        (r"/snap_servers.json", SnapServersHandler),
        (r"/mopidy_servers.json", MopidyServersHandler),
        (r"/client", ClientSettingsHandler),
        (r"/browse.json", BrowseHandler),
        (r"/play", PlayHandler),
        (r"/stop", MopidyStopPlaybackHandler),
        (r"/(.*)", StaticFileHandler, {
            'path': os.path.join(os.path.dirname(__file__), 'frontend-react', 'build'),
            'default_filename': 'index.html'
        }),
    ], debug=debug)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Snapcast control')
    parser.add_argument("--debug", help="run tornado in debug mode", action="store_true")
    parser.add_argument("--loglevel", help="loglevel", default='DEBUG')
    parser.add_argument("--port", help="web server port", default=8080, type=int)
    args = parser.parse_args()

    logging_file_handler = logging.FileHandler("server.log", encoding='utf-8')
    logging_file_handler.setLevel(logging.INFO)
    logging_file_handler.addFilter(lambda r: not (r.name == "tornado.access"))
    logging_stdout_handler = logging.StreamHandler(sys.stdout)
    logging.basicConfig(
        level=getattr(logging, args.loglevel.upper()),
        handlers=[logging_file_handler, logging_stdout_handler],
        datefmt='%Y-%m-%d %H:%M:%S',
        format='[%(asctime)s] [%(name)s] %(levelname)s: %(message)s'
    )

    AsyncIOMainLoop().install()
    ioloop = asyncio.get_event_loop()

    zero_snap_servers = []
    zero_mopidy_servers = []
    snap_servers = {}

    def on_add_snapserver(info: ServiceInfo):
        snap_server = Snapserver(ioloop, host=info.parsed_addresses()[0],
                                 port=info.port, reconnect=True)
        snap_servers[info.name.replace('.' + SNAPCAST_ZERO_NAME, '')] = snap_server
        ioloop.create_task(snap_server.start())

    def on_remove_snapserver(info: ServiceInfo):
        snap_servers.pop(info.name.replace('.' + SNAPCAST_ZERO_NAME, ''))

    zeroconf = Zeroconf()
    zeroconf.add_service_listener('_mopidy-http._tcp.local.', ZeroListener(zero_mopidy_servers))
    zeroconf.add_service_listener(SNAPCAST_ZERO_NAME, ZeroListener(zero_snap_servers,
                                                                   on_add_snapserver,
                                                                   on_remove_snapserver))

    @asyncio.coroutine
    def sync_snapserver():
        while True:
            logging.info('Synchronizing snapserver')
            for server in snap_servers.values():
                status = yield from server.status()
                server.synchronize(status)
            yield from asyncio.sleep(60)

    ioloop.create_task(sync_snapserver())

    http_client = AsyncHTTPClient()

    logging.info("Starting web app")
    app = make_app(args.debug)
    app.listen(args.port)
    ioloop.run_forever()
