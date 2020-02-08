export interface Client {
  id: string,
  volume: number,
  latency: number,
  muted: boolean,
  connected: boolean,
  stream: string
}

export interface Stream {
  id: string,
  status: "playing" | "idle",
  meta: {
    TITLE?: string,
    ARTIST?: string,
    ALBUM?: string,
    COVER?: string
  }
}

export interface MopidyServer {
  name: string
}

export interface BibItem {
  name: string,
  uri: string,
  type: string,
}

export interface SnapServer {
  clients: Client[],
  streams: Stream[]
}