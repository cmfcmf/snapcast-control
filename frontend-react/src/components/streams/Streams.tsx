import React, { useEffect, useState } from "react";
import "./Streams.css";
import { Client, Stream, MopidyServer, SnapServer } from "../../entities";
import { Link } from "react-router-dom";
import clsx from "clsx";
import { useIsAdmin, times } from "../../util";
import { interval } from "rxjs";
import { exhaustMap, filter } from "rxjs/operators";
import { Api } from "../../api";
import ContentLoader from 'react-content-loader'

const ClientColumn = ({ serverName, client, streams, mopidyServers }: { serverName: string, client: Client, streams: Stream[], mopidyServers: MopidyServer[] }) => {
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
          :
          streams.map(stream => <React.Fragment key={stream.id}>
            {(stream.status === 'playing' || client.stream === stream.id || stream.id.startsWith(client.id)) &&
              <button onClick={() => Api.setStream(serverName, client, stream)} className={clsx("list-group-item", "list-group-item-action", {
                'active': client.stream === stream.id,
                'list-group-item-light': !stream.id.startsWith(client.id) && stream.status !== "playing"
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
          </React.Fragment>)

        }
      </div>{client.connected && client.muted &&
        <div className="list-group-item list-group-item-danger">
          Ton ist ausgeschaltet!
  </div>}
      {(client.connected || isAdmin) &&
        <div className="card-body">
          {client.connected && client.muted && <button onClick={() => Api.unmute(serverName, client)} className="btn btn-link p-0 card-link">Ton einschalten</button>}
          {client.connected && !client.muted && <button onClick={() => Api.mute(serverName, client)} className="btn btn-link p-0 card-link">Ton ausschalten</button>}
          {isAdmin && <button onClick={() => Api.delete(serverName, client)} className="btn btn-link p-0 card-link text-danger">Löschen</button>}
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
              <button className="btn btn-secondary" type="submit" onClick={() => Api.setLatency(serverName, client, latency)}>Speichern</button>
            </span>
          </div>
        </div>}
    </div></>
}

const StreamSkeleton = () => {
  const rows = 6;
  const rowHeight = 40;
  const rowPadding = 5;

  const height = (rows + 1) * (rowHeight + rowPadding);

  return <div className="row">
    <div className="col-12">
      <ContentLoader
        speed={2}
        width={350}
        height={50}
        viewBox={`0 0 ${350} ${50}`}
        backgroundColor="#f3f3f3"
        foregroundColor="#ecebeb"
      >
        <rect x={0} y={0} rx="5" ry="5" width={350} height={40} />
      </ContentLoader>
    </div>
    {times(4, i =>
      <div className="col-12 col-sm-6 col-lg-4 col-xl-3" key={i}>
        <ContentLoader
          speed={2}
          width="100%"
          height={height}
          viewBox={`0 0 100 ${height}`}
          backgroundColor="#f3f3f3"
          foregroundColor="#ecebeb"
          preserveAspectRatio="none"
        >
          {times(rows, i =>
            <rect key={i} x={0} y={(i > 0 ? i + 1 : 0) * (rowHeight + rowPadding)} rx="5" ry="5" width="100%" height={i === 0 ? rowHeight * 2 : rowHeight} />
          )}
        </ContentLoader>
      </div>)}
  </div>
}

const Streams = () => {
  const [servers, setServers] = useState<[string, SnapServer][]>([]);
  const [mopidyServers, setMopidyServers] = useState<MopidyServer[]>([]);
  const [state, setLoadingState] = useState<"loading" | "done">("loading");

  useEffect(() => {
    const subscription = interval(2000)
      .pipe(
        // Do not constantly update if the window is hidden
        filter(() => !document.hidden),
        exhaustMap(async () => {
          try {
            const servers = Object.entries(await Api.getSnapServers());
            const mopidyServers = await Api.getMopidyServers();

            servers.sort(([nameA], [nameB]) => nameA.localeCompare(nameB));
            servers.forEach(([name, server]) => {
              server.clients.sort((a: Client, b: Client) => {
                if (a.connected && !b.connected) {
                  return -1;
                }
                if (!a.connected && b.connected) {
                  return 1;
                }
                return a.id.localeCompare(b.id);
              });
              server.streams.sort((a: Stream, b: Stream) => {
                if (a.status === "playing" && b.status === "idle") {
                  return -1;
                }
                if (a.status === "idle" && b.status === "playing") {
                  return 1;
                }
                return a.id.localeCompare(b.id);
              });
            });
            setServers(servers);
            setMopidyServers(mopidyServers);
            setLoadingState("done");
          } catch (e) {
            console.error(e);
            setLoadingState("loading");
          }
        })
      )
      .subscribe();
    return () => subscription.unsubscribe();
  }, []);

  return <>{state === "loading"
    ? <StreamSkeleton />
    : <>
        {servers.map(([name, server]) =>
          <div className="row" key={name}>
            {servers.length > 1 && <div className="col-12"><h2>{name}</h2></div>}
            {
              server.clients.map(client =>
                <div className="col-12 col-sm-6 col-lg-4 col-xl-3" key={client.id}>
                  <ClientColumn serverName={name} client={client} streams={server.streams} mopidyServers={mopidyServers} />
                </div>
              )
            }
          </div>
        )}
        {servers.length === 0 && <div className="alert alert-warning">Keine SnapCast Instanzen auffindbar. Suche weiter...</div>}
    </>}</>
}

export default Streams;