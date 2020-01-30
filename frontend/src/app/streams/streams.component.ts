import {Component, OnDestroy, OnInit} from '@angular/core';
import {ApiService} from '../api.service';
import {Stream} from '../stream';
import {Client} from '../client';
import {MopidyServer} from '../mopidy-server';
import {ActivatedRoute, ParamMap} from '@angular/router';
import 'rxjs/add/operator/filter';
import {TimerObservable} from 'rxjs/observable/TimerObservable';
import 'rxjs/add/operator/takeWhile';
import 'rxjs/add/operator/first';

@Component({
  selector: 'app-streams',
  templateUrl: './streams.component.html',
  styleUrls: ['./streams.component.css']
})
export class StreamsComponent implements OnInit, OnDestroy {
  streams: Stream[] = [];
  clients: Client[] = [];
  mopidyServers: MopidyServer[] = [];
  isAdmin: boolean;

  private refreshInterval = 2000;
  private alive: boolean;

  constructor(
    public api: ApiService,
    private activatedRoute: ActivatedRoute
  ) { }

  ngOnInit(): void {
    this.activatedRoute.queryParamMap.subscribe((params: ParamMap) => {
      this.isAdmin = ['true', '1'].indexOf(params.get('is_admin')) > -1;
    });
    this.alive = true;

    TimerObservable.create(0, this.refreshInterval)
      .takeWhile(() => this.alive)
      .subscribe(() => {
        this.api.getClients().subscribe((clients: Client[]) => {
          this.clients = clients.sort((a: Client, b: Client) => {
            if (a.connected && !b.connected) {
              return -1;
            }
            if (!a.connected && b.connected) {
              return 1;
            }
            return 0;
          });
        });
        this.api.getStreams().subscribe((streams: Stream[]) => {
          this.streams = streams;
        });
        this.api.getMopidyServers()
          .first()
          .subscribe((mopidyServers: MopidyServer[]) => {
            this.mopidyServers = mopidyServers;
          });
      });
  }

  public getMopidyServerNameForClient(client: Client): string|null {
    const mopidyServer = this.mopidyServers.find((each: MopidyServer, index: number) => {
      return each.name.startsWith(client.id);
    });

    return mopidyServer !== undefined ? mopidyServer.name : null;
  }

  ngOnDestroy(): void {
    this.alive = false;
  }
}
