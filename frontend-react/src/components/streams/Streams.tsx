import React, { useEffect, useState } from "react";
import "./Streams.css";
import { Client, Stream, MopidyServer } from "../../entities";
import { Link } from "react-router-dom";
import clsx from "clsx";
import { useIsAdmin, times } from "../../util";
import { interval } from "rxjs";
import { exhaustMap, filter } from "rxjs/operators";
import { Api } from "../../api";
import ContentLoader from 'react-content-loader'

const ClientColumn = ({ client, streams, mopidyServers }: { client: Client, streams: Stream[], mopidyServers: MopidyServer[] }) => {
  const isAdmin = useIsAdmin();
  const [latency, setLatency] = useState(client.latency);

  function getMopidyServerNameForClient(client: Client): string | null {
    const mopidyServer = mopidyServers.find((each: MopidyServer) => each.name.startsWith(client.id));
    return mopidyServer !== undefined ? mopidyServer.name : null;
  }

  const mopidyServerName = getMopidyServerNameForClient(client);

  return <>
    <div className="card mb-2">
      <div className="card-body">
        <h4 className="card-title">{client.id}</h4>
        {isAdmin && <h6 className="card-subtitle mb-2 text-muted">Latenz: {client.latency}</h6>}
      </div>
      <div className="list-group list-group-flush">
        {!client.connected
          ? <div className="list-group-item list-group-item-danger">
            Nicht verbunden
        </div>
          : <div>
            {/*{% for stream in sorted(server.streams, key=lambda stream: (stream.status != 'playing', not stream.friendly_name.startswith(client.identifier), stream.friendly_name)) %} */}
            {streams.map(stream => <React.Fragment key={stream.id}>
              {(stream.status === 'playing' || client.stream === stream.id || stream.id.startsWith(client.id)) &&
                <button onClick={() => Api.setStream(client, stream)} className={clsx("list-group-item", "list-group-item-action", {
                  'active': client.stream === stream.id,
                  'list-group-item-info': stream.id.startsWith(client.id)
                })}>
                  <i className={clsx("icon-stream-idle", {
                    'icon-stream-idle': stream.status === 'idle',
                    'icon-stream-playing': stream.status === 'playing'
                  })} />
                  {stream.status === 'playing' ? <strong>{stream.id}</strong> : <>{stream.id}</>}
                  <br />
                  {stream.status === 'playing' &&
                    <div className="clearfix mt-1">
                      {stream.meta.COVER && stream.meta.COVER.length && <img src={'data:image/png;base64,' + stream.meta.COVER} className="cover ml-2 float-right" alt="" />}
                      <em>{stream.meta.TITLE} {stream.meta.ARTIST ? ' - ' + stream.meta.ARTIST : ''}</em>
                    </div>}
                </button>}
            </React.Fragment>)}
          </div>
        }
      </div>{client.connected && client.muted &&
        <div className="list-group-item list-group-item-danger">
          Ton ist ausgeschaltet!
  </div>}
      {(client.connected || isAdmin) &&
        <div className="card-body">
          {client.connected && client.muted && <button onClick={() => Api.unmute(client)} className="btn btn-link p-0 card-link">Ton einschalten</button>}
          {client.connected && !client.muted && <button onClick={() => Api.mute(client)} className="btn btn-link p-0 card-link">Ton ausschalten</button>}
          {isAdmin && <button onClick={() => Api.delete(client)} className="btn btn-link p-0 card-link text-danger">Löschen</button>}
        </div>}
      {client.connected && mopidyServerName !== null &&
        <div className="card-body">
          <Link to={`/browse?name=${encodeURIComponent(mopidyServerName)}&play=0`} className="card-link">Lokale Musik ändern</Link>
        </div>}
      {isAdmin &&
        <div className="card-body">
          <div className="input-group">
            <input type="number" min={0} name="latency" value={latency} onChange={(event) => setLatency(parseInt(event.target.value))} className="form-control" placeholder="Latenz" aria-label="Latenz" />
            <span className="input-group-append">
              <button className="btn btn-secondary" type="submit" onClick={() => Api.setLatency(client, latency)}>Speichern</button>
            </span>
          </div>
        </div>}
    </div></>
}

const StreamSkeleton = () => {
  const rows = 6;
  const rowHeight = 40;
  const rowWidth = 250;
  const rowPadding = 5;

  const height = (rows + 1) * (rowHeight + rowPadding);

  return <ContentLoader
    speed={2}
    width={"100%"}
    height={height}
    viewBox={`0 0 ${rowWidth} ${height}`}
    backgroundColor="#f3f3f3"
    foregroundColor="#ecebeb"
  >
    {times(rows, i =>
      <rect key={i} x={0} y={(i > 0 ? i + 1 : 0) * (rowHeight + rowPadding)} rx="5" ry="5" width={rowWidth} height={i === 0 ? rowHeight * 2 : rowHeight} />
    )}
  </ContentLoader>;
}

const Streams = () => {
  const [streams, setStreams] = useState<Stream[]>([]);
  const [clients, setClients] = useState<Client[]>([]);
  const [mopidyServers, setMopidyServers] = useState<MopidyServer[]>([]);
  const [state, setLoadingState] = useState<"loading" | "done">("loading");

  useEffect(() => {
    const subscription = interval(2000)
      .pipe(
        // Do not constantly update if the window is hidden
        filter(() => !document.hidden),
        exhaustMap(async () => {
          try {
            const clients = await Api.getClients();
            const streams = await Api.getStreams();
            const mopidyServers = await Api.getMopidyServers();

            setClients(clients.sort((a: Client, b: Client) => {
              if (a.connected && !b.connected) {
                return -1;
              }
              if (!a.connected && b.connected) {
                return 1;
              }
              return 0;
            }));
            setStreams(streams);
            setMopidyServers(mopidyServers);
            setLoadingState("done");
          } catch (e) {
            setLoadingState("loading");
          }
        })
      )
      .subscribe();
    return () => subscription.unsubscribe();
  }, []);

  return <div className="row">
    {state === "loading"
      ? times(4, i =>
        <div className="col-12 col-sm-6 col-lg-4 col-xl-3" key={i}>
          <StreamSkeleton />
        </div>)
      : clients.map(client =>
        <div className="col-12 col-sm-6 col-lg-4 col-xl-3" key={client.id}>
          <ClientColumn client={client} streams={streams} mopidyServers={mopidyServers} />
        </div>
      )}

  </div>
}

export default Streams;