import json

import tornado.ioloop
import tornado.gen
import tornado.web
from snapcast.control import Snapserver
from tornado.escape import utf8
from tornado.escape import url_escape
from tornado.httpclient import AsyncHTTPClient
from tornado.platform.asyncio import AsyncIOMainLoop
import asyncio
from functools import reduce

SNAPSERVER_HOST = 'musik.local'
MOPIDY_RPC_URL = 'http://musik.local:6680/mopidy/rpc'


class BaseHandler(tornado.web.RequestHandler):
    def initialize(self):
        super().initialize()

    def get_template_namespace(self):
        namespace = super().get_template_namespace()
        namespace['connected'] = server._protocol is not None
        namespace['is_admin'] = self.get_argument('is_admin', False)
        return namespace

    async def mopidy_rpc_request(self, method, params={}):
        body = json.dumps({
            "method": method,
            "jsonrpc": "2.0",
            "params": params,
            "id": 1
        })
        headers = dict()
        headers['Content-Type'] = 'application/json'
        response = await http_client.fetch(MOPIDY_RPC_URL, method='POST', body=body, headers=headers)
        return json.loads(utf8(response.body))['result']


class MainHandler(BaseHandler):
    def get(self):
        self.redirect("/browse")


class BrowseHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        uri = self.get_argument('uri', None)

        items = yield self.mopidy_rpc_request("core.library.browse", {'uri': uri})
        tracks = list(filter(lambda item: item['type'] == 'track', items))
        if len(tracks) > 0:
            track_uris = reduce(lambda uris, track: uris + '&uri=' + url_escape(track['uri']), tracks, '')
        else:
            track_uris = False
        self.render("browse.html", title="Musik", items=items, track_uris=track_uris)


class PlayHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        uris = self.get_arguments('uri')
        yield self.mopidy_rpc_request("core.tracklist.clear")
        tracks = yield self.mopidy_rpc_request("core.tracklist.add", {'uris': uris})
        yield self.mopidy_rpc_request("core.playback.play", {'tlid': tracks[0]['tlid']})
        self.redirect("/streams")


class StreamsHandler(BaseHandler):
    def get(self):
        self.render("streams.html", title="Streams", clients=sorted(server.clients, key=lambda client: client.identifier), server=server)


class ClientSettingsHandler(BaseHandler):
    @tornado.gen.coroutine
    def get(self):
        client_id = self.get_argument('id')
        action = self.get_argument('action')

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
            print('Unknown action!')
            pass

        self.redirect('/streams')


def make_app():
    return tornado.web.Application([
        (r"/", MainHandler),
        (r"/streams", StreamsHandler),
        (r"/client", ClientSettingsHandler),
        (r"/browse", BrowseHandler),
        (r"/play", PlayHandler),
    ], debug=True)


if __name__ == "__main__":
    AsyncIOMainLoop().install()
    ioloop = asyncio.get_event_loop()

    server = Snapserver(ioloop, SNAPSERVER_HOST, reconnect=True)
    ioloop.run_until_complete(server.start())

    http_client = AsyncHTTPClient()

    app = make_app()
    app.listen(8083)
    ioloop.run_forever()