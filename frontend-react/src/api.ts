import { Stream, Client, MopidyServer, BibItem } from "./entities";

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
    const result = await fetch(url.href, { signal: controller.signal });
    return await result.json();
  }

  public static async getStreams(){
    return this.makeRequest<Stream[]>('/streams.json');
  }

  public static async getClients() {
    return this.makeRequest<Client[]>('/clients.json');
  }

  public static async getMopidyServers() {
    return this.makeRequest<MopidyServer[]>('/mopidy_servers.json');
  }

  public static async mute(client: Client) {
    await this.setClientConfig(client, 'mute');
  }

  public static async unmute(client: Client) {
    await this.setClientConfig(client, 'unmute');
  }

  public static async delete(client: Client) {
    await this.setClientConfig(client, 'delete');
  }

  public static async setLatency(client: Client, latency: number) {
    await this.setClientConfig(client, 'set_latency', {latency: latency});
  }

  public static async setStream(client: Client, stream: Stream) {
    await this.setClientConfig(client, 'set_stream', {stream: stream.id});
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

  private static async setClientConfig(client: Client, action: string, params = {}) {
    return this.makeRequest('/client', { id: client.id, action, ...params });
  }
}
