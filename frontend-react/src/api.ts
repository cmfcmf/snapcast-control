import { Stream, Client, MopidyServer, BibItem, SnapServer } from "./entities";

export class Api {
  private static async makeRequest<T>(path: string, params: Record<string, string|string[]> = {}): Promise<T> {
    const url = new URL(path, document.baseURI);
    for (const [key, values] of Object.entries(params)) {
      if (Array.isArray(values)) {
        values.forEach(value => url.searchParams.append(key, value));
      } else {
        url.searchParams.append(key, values);
      }
    }

    const controller = new AbortController();
    setTimeout(() => controller.abort(), 5000);
    const result = await fetch(url.href, { signal: controller.signal, redirect: "error" });
    return await result.json();
  }

  public static async getSnapServers(){
    return this.makeRequest<Record<string, SnapServer>>('/snap_servers.json');
  }

  public static async getMopidyServers() {
    return this.makeRequest<MopidyServer[]>('/mopidy_servers.json');
  }

  public static async mute(serverName: string, client: Client) {
    await this.setClientConfig(serverName, client, 'mute');
  }

  public static async unmute(serverName: string, client: Client) {
    await this.setClientConfig(serverName, client, 'unmute');
  }

  public static async delete(serverName: string, client: Client) {
    await this.setClientConfig(serverName, client, 'delete');
  }

  public static async setLatency(serverName: string, client: Client, latency: number) {
    await this.setClientConfig(serverName, client, 'set_latency', {latency: latency});
  }

  public static async setStream(serverName: string, client: Client, stream: Stream) {
    await this.setClientConfig(serverName, client, 'set_stream', {stream: stream.id});
  }

  public static browse(uri: string|null, mopidyServerName: string): Promise<BibItem[]> {
    const params: Record<string, string> = {
      name: mopidyServerName
    };
    if (uri !== null) {
      params['uri'] = uri;
    }
    return this.makeRequest<BibItem[]>('/browse.json', params);
  }

  public static async play(uris: string[], mopidyServerName: string) {
    return this.makeRequest('/play', {
        name: mopidyServerName,
        uri: uris
      })
    }

  public static async stop(mopidyServerName: string) {
    return this.makeRequest('/stop', { name: mopidyServerName });
  }

  private static async setClientConfig(serverName: string, client: Client, action: string, params = {}) {
    return this.makeRequest('/client', { id: client.id, action, server_name: serverName, ...params });
  }
}
