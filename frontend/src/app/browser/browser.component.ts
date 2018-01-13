import { Component, OnInit } from '@angular/core';
import {ActivatedRoute, ParamMap, Router} from '@angular/router';
import {BibItem} from '../bib-item';
import {ApiService} from '../api.service';

@Component({
  selector: 'app-browser',
  templateUrl: './browser.component.html',
  styleUrls: ['./browser.component.css']
})
export class BrowserComponent implements OnInit {
  items: BibItem[] = [];
  mopidyServerName: string;

  constructor(
    private activatedRoute: ActivatedRoute,
    private api: ApiService,
    private router: Router
  ) { }

  ngOnInit() {
    this.activatedRoute.queryParamMap.subscribe((params: ParamMap) => {
      const uri = params.get('uri');
      this.mopidyServerName = params.get('name');
      this.api.browse(uri, this.mopidyServerName).subscribe((items: BibItem[]) => {
        this.items = items;
      });
    });
  }

  public playUris(uris: string[]) {
    this.api.play(uris, this.mopidyServerName);
    this.router.navigateByUrl('/');
  }

  public getTrackUris(): string[] {
    return this.items
      .filter((item: BibItem) => item.type === 'track')
      .map((item: BibItem) => item.uri);
  }
}
