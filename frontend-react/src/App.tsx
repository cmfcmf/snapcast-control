import React from 'react';
import './App.css';
import {
  Switch,
  Route,
  Link,
  useLocation
} from "react-router-dom";
import clsx from "clsx";
import Streams from './components/streams/Streams';
import { useIsAdmin } from './util';
import Browser from './components/browser/browser';

const App = () => {
  const isRoot = useLocation().pathname === "/";
  const isAdmin = useIsAdmin();
  return (
    <>
        <nav className="navbar navbar-expand-lg navbar-light bg-light mb-2">
          <div className="container">
            <Link to="/" className="navbar-brand mb-0 h1">Musik</Link>
            <ul className="navbar-nav mr-auto">
              <li className={clsx("nav-item", {
                active: isAdmin && isRoot
              })}>
                <Link to="/?is_admin=1" className="nav-link">Admin</Link>
              </li>
            </ul>
          </div>
        </nav>
        <div className="container">
          <Switch>
            <Route exact path="/">
              <Streams />
            </Route>
            <Route exact path="/browse">
              <Browser />
            </Route>
          </Switch>
        </div>
    </>
  );
}

export default App;
