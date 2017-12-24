#!/usr/bin/python3
import json
import argparse
import sys
import time
import logging
import os

import tornado.ioloop
import tornado.gen
import tornado.web
from snapcast.control import Snapserver
from tornado.escape import to_unicode
from tornado.escape import url_escape
from tornado.httpclient import AsyncHTTPClient
from tornado.platform.asyncio import AsyncIOMainLoop
import asyncio
from functools import reduce
from zeroconf import Zeroconf


class BaseHandler(tornado.web.RequestHandler):
    def initialize(self):
        super().initialize()

    def get_template_namespace(self):
        namespace = super().get_template_namespace()
        namespace['connected'] = server._protocol is not None
        namespace['is_admin'] = self.get_argument('is_admin', False)
        namespace['debug'] = self.get_argument('debug', False)
        return namespace

    async def mopidy_rpc_request(self, server_name, method, params={}):
        body = json.dumps({
            "method": method,
            "jsonrpc": "2.0",
            "params": params,
            "id": 1
        })

        headers = dict()
        headers['Content-Type'] = 'application/json'

        mopidy_server = self.get_mopidy_server_from_name(server_name)
        url = 'http://{}:{}/mopidy/rpc'.format(mopidy_server.server, mopidy_server.port)

        response = await http_client.fetch(url, method='POST', body=body, headers=headers)

        return json.loads(to_unicode(response.body))['result']

    def get_mopidy_server_from_name(self, name):
        return list(filter(lambda mopidy_server: mopidy_server.name == name, mopidy_servers))[0]


class BrowseHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        uri = self.get_argument('uri', None)
        name = self.get_argument('name')

        items = yield self.mopidy_rpc_request(name, "core.library.browse", {'uri': uri})
        tracks = list(filter(lambda item: item['type'] == 'track', items))
        if len(tracks) > 0:
            track_uris = reduce(lambda uris, track: uris + '&uri=' + url_escape(track['uri']), tracks, '')
        else:
            track_uris = False

        self.render("browse.html", title="Musik", items=items, track_uris=track_uris, name=name)


class PlayHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        name = self.get_argument('name')
        uris = self.get_arguments('uri')
        yield self.mopidy_rpc_request(name, "core.tracklist.clear")
        tracks = yield self.mopidy_rpc_request(name, "core.tracklist.add", {'uris': uris})
        yield self.mopidy_rpc_request(name, "core.playback.play", {'tlid': tracks[0]['tlid']})
        self.redirect("/")


class StreamsHandler(BaseHandler):
    def get(self):
        self.render(
            "streams.html",
            title="Streams",
            clients=sorted(server.clients, key=lambda client: client.identifier),
            server=server,
            mopidy_servers=mopidy_servers,
        )


class ClientSettingsHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        client_id = self.get_argument('id')
        action = self.get_argument('action')
        is_admin = self.get_argument('is_admin', False)

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

        self.redirect('/?is_admin=1' if is_admin else '/')


def make_app(debug):
    return tornado.web.Application([
        (r"/", StreamsHandler),
        (r"/client", ClientSettingsHandler),
        (r"/browse", BrowseHandler),
        (r"/play", PlayHandler),
        (r"/(.*)", tornado.web.StaticFileHandler, {'path': (os.path.join(os.path.dirname(__file__), 'static'))}),
    ], debug=debug)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Snapcast control')
    parser.add_argument("--debug", help="run tornado in debug mode", action="store_true")
    parser.add_argument("--loglevel", help="loglevel", default='DEBUG')
    parser.add_argument("--port", help="web server port", default=8080, type=int)
    args = parser.parse_args()

    logging.basicConfig(level=getattr(logging, args.loglevel.upper()))

    snap_servers = []
    mopidy_servers = []

    class MopidyListener:

        def add_service(self, zeroconf, type, name):
            info = zeroconf.get_service_info(type, name)
            logging.debug("Service %s added, service info: %s" % (name, info))
            global mopidy_servers
            mopidy_servers.append(info)

        def remove_service(self, zeroconf, type, name):
            logging.warning("Service %s removed" % name)
            global mopidy_servers
            mopidy_servers = list(filter(lambda mopidy_server: mopidy_server.name != name, mopidy_servers))


    class SnapListener:
        def add_service(self, zeroconf, type, name):
            info = zeroconf.get_service_info(type, name)
            logging.debug("Service %s added, service info: %s" % (name, info))
            global snap_servers
            snap_servers.append(info)

        def remove_service(self, zeroconf, type, name):
            logging.warning("Service %s removed" % name)
            global snap_servers
            snap_servers = list(filter(lambda snap_server: snap_server.name != name, snap_servers))

    zeroconf = Zeroconf()
    zeroconf.add_service_listener('_mopidy-http._tcp.local.', MopidyListener())
    # TODO: This should use _snapcast-jsonrpc._tcp.local.
    #       However, this name does not comply with https://tools.ietf.org/html/rfc6763#section-7.2
    #       and is therefore rejected by the zeroconf library.
    zeroconf.add_service_listener('_snapcast._tcp.local.', SnapListener())

    logging.info("Discovering services")
    while len(snap_servers) == 0:
        time.sleep(0.1)

    if len(snap_servers) != 1:
        logging.error("Exactly 1 snapserver expected, found {}.".format(len(snap_servers)))
        sys.exit(1)
    snap_server = snap_servers[0]

    AsyncIOMainLoop().install()
    ioloop = asyncio.get_event_loop()

    logging.info("Connecting to snapserver")
    # TODO: This should also specify port=snap_server.port
    #       However, we first need to fix the bug above.
    server = Snapserver(ioloop, host=snap_server.server, reconnect=True)
    ioloop.run_until_complete(server.start())

    http_client = AsyncHTTPClient()

    logging.info("Starting web app")
    app = make_app(args.debug)
    app.listen(args.port)
    ioloop.run_forever()
