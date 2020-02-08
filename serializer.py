import collections

from snapcast.control.client import Snapclient
from snapcast.control.stream import Snapstream
from zeroconf import ServiceInfo


class Serializer(object):
    def serialize(self, data):
        if isinstance(data, list):
            return list(map(self.serialize, data))
        if isinstance(data, dict):
            return {key: self.serialize(value) for key, value in data.items()}
        if isinstance(data, Snapclient):
            return self._serialize_snapclient(data)
        if isinstance(data, Snapstream):
            return self._serialize_snapstream(data)
        if isinstance(data, ServiceInfo):
            return self._serialize_serviceinfo(data)

        return data

    @staticmethod
    def _serialize_snapclient(snap_client):
        return {
            'id': snap_client.identifier,
            'muted': snap_client.muted,
            'volume': snap_client.volume,
            'name': snap_client.friendly_name,
            'latency': snap_client.latency,
            'connected': snap_client.connected,
            'stream': snap_client.group.stream if snap_client.group else ''
        }

    @staticmethod
    def _serialize_serviceinfo(service_info):
        return {
            'name': service_info.name
        }

    @staticmethod
    def _serialize_snapstream(snap_stream: Snapstream):
        return {
            'id': snap_stream.identifier,
            'status': snap_stream.status,
            'meta': snap_stream._stream['meta']
        }
