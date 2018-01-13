export class Client {
  constructor(
    public id: string,
    public volume: number,
    public latency: number,
    public muted: boolean,
    public connected: boolean,
    public stream: string) {}
}
