import { Link, useHistory } from "react-router-dom";
import { Api } from "../../api";
import { BibItem } from "../../entities";
import { useQuery } from "../../util";
import { useState, useEffect } from "react";
import React from "react";

const Browser = () => {
  const history = useHistory();
  const query = useQuery();
  const uri = query.get('uri');
  const isRoot = !uri;
  const mopidyServerName = query.get('name')!;

  const [items, setItems] = useState<BibItem[]>([]);
  const [state, setLoadingState] = useState<"loading" | "done" | "failed">("loading");

  useEffect(() => {
    setLoadingState("loading");
    Api.browse(uri, mopidyServerName)
      .then(items => {
        setItems(items);
        setLoadingState("done");
      })
      .catch(err => {
        console.log(err);
        setItems([]);
        setLoadingState("failed");
      });
  }, [uri, mopidyServerName]);

  function stop() {
    Api.stop(mopidyServerName);
    history.push("/");
  }

  function playUris(uris: string[]) {
    Api.play(uris, mopidyServerName);
    history.push("/");
  }

  const trackUris = items
    .filter(item => item.type === 'track')
    .map(item => item.uri);

  return (
    <div className="col">
      <div className="list-group">
        {isRoot && <button className="list-group-item list-group-item-action" onClick={() => stop()}>
          Playback stoppen
        </button>}
        {trackUris.length > 0 && <button className="list-group-item list-group-item-action active" onClick={() => playUris(trackUris)}>
          Alle abspielen
        </button>}
        {state === "loading" && <div className="list-group-item">Lade...</div>}
        {state === "failed" && <div className="list-group-item list-group-item-danger">
            Daten konnten nicht geladen werden.
          </div>}
        {state === "done" && items.map(item => <React.Fragment key={item.uri}>
          {item.type === 'track' ? <button onClick={() => playUris([item['uri']])} className="list-group-item list-group-item-action">
            {item.name}
          </button> :
            <Link to={`/browse?uri=${encodeURIComponent(item.uri)}&name=${encodeURIComponent(mopidyServerName)}`} className="list-group-item list-group-item-action">
              {item.name}
            </Link>}
        </React.Fragment>)}
      </div>
    </div>
  );
}

export default Browser;