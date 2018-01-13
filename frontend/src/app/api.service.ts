import { Injectable } from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {Observable} from 'rxjs/Observable';
import {Stream} from './stream';
import {catchError} from 'rxjs/operators';
import {of} from 'rxjs/observable/of';
import {BibItem} from './bib-item';
import {Client} from './client';
import {MopidyServer} from './mopidy-server';

@Injectable()
export class ApiService {
  baseUrl = '';

  constructor(
    private http: HttpClient,
  ) { }

  public getStreams(): Observable<Stream[]> {
    return this.http.get<Stream[]>(this.baseUrl + '/streams.json')
      .pipe(
        catchError(this.handleError('getStreams', []))
      );
  }

  public getClients(): Observable<Client[]> {
    return this.http.get<Client[]>(this.baseUrl + '/clients.json')
      .pipe(
        catchError(this.handleError('getClients', []))
      );
  }

  public getMopidyServers(): Observable<MopidyServer[]> {
    return this.http.get<MopidyServer[]>(this.baseUrl + '/mopidy_servers.json')
      .pipe(
        catchError(this.handleError('getMopidyServers', []))
      );
  }

  public mute(client: Client): void {
    this.setClientConfig(client, 'mute');
  }

  public unmute(client: Client): void {
    this.setClientConfig(client, 'unmute');
  }

  public delete(client: Client): void {
    this.setClientConfig(client, 'delete');
  }

  public setLatency(client: Client, latency: number): void {
    this.setClientConfig(client, 'set_latency', {latency: latency});
  }

  public setStream(client: Client, stream: Stream): void {
    this.setClientConfig(client, 'set_stream', {stream: stream.id});
  }

  public browse(uri: string|null, mopidyServerName: string): Observable<BibItem[]> {
    const params = {
      name: mopidyServerName
    };
    if (uri !== null) {
      params['uri'] = uri;
    }
    return this.http.get<BibItem[]>(this.baseUrl + '/browse.json', {
      params: params
    })
      .pipe(
        catchError(this.handleError('browse', []))
      );
  }

  public play(uris: string[], mopidyServerName: string): void {
    this.http.get(this.baseUrl + '/play', {params: {
        name: mopidyServerName,
        uri: uris
      }})
      .pipe(
        catchError(this.handleError('play', []))
      )
      .subscribe();
  }

  private setClientConfig(client: Client, action: string, params = {}) {
    params['id'] = client.id;
    params['action'] = action;

    this.http.get(this.baseUrl + '/client', {params: params})
      .pipe(
        catchError(this.handleError('setClientConfig', []))
      )
      .subscribe();
  }

  /**
   * Handle Http operation that failed.
   * Let the app continue.
   * @param operation - name of the operation that failed
   * @param result - optional value to return as the observable result
   */
  private handleError<T> (operation: string, result?: T) {
    return (error: any): Observable<T> => {
      console.error(error);
      console.error(`${operation} failed: ${error.message}`);

      // Let the app keep running by returning an empty result.
      return of(result as T);
    };
  }
}
