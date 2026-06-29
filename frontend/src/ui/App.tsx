/**
 * @layer ui
 * @description Root application layout with CSS Grid shell.
 * @depends ui/layout/TopBar, ui/layout/MainPane, ui/layout/SidePane, ui/layout/StatusBar
 * @must-not Contain business logic. Import from graph/.
 */

import { useEffect } from 'react';
import { MantineProvider, createTheme } from '@mantine/core';
import '@mantine/core/styles.css';
import '@mantine/dates/styles.css';
import './App.css';
import { TopBar } from './layout/TopBar';
import { MainPane } from './layout/MainPane';
import { SidePane } from './layout/SidePane';
import { StatusBar } from './layout/StatusBar';
import { loadFromApi } from '../core/actions';

const theme = createTheme({
  primaryColor: 'blue',
  defaultRadius: 'sm',
});

export default function App() {
  useEffect(() => {
    loadFromApi();
  }, []);

  return (
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <div className="app-shell">
        <div className="app-topbar">
          <TopBar />
        </div>
        <div className="app-main">
          <MainPane />
        </div>
        <div className="app-side">
          <SidePane />
        </div>
        <div className="app-statusbar">
          <StatusBar />
        </div>
      </div>
    </MantineProvider>
  );
}
