import { NgModule } from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {StreamsComponent} from './streams/streams.component';
import {BrowserComponent} from './browser/browser.component';


const routes: Routes = [
  { path: '', component: StreamsComponent },
  { path: 'browse', component: BrowserComponent },
];

@NgModule({
  exports: [RouterModule],
  imports: [ RouterModule.forRoot(routes) ],
})
export class AppRoutingModule { }
