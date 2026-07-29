/**
 * package: main / entry
 * type:    entry
 * job:     mount React, which renders the shell
 * limits:  wiring only; the store and the renderer own their own lifecycles
 *
 * Entry point. Mounts React, which renders the shell; the graph store and the
 * Sigma instance are created by core and render/, outside React entirely.
 */

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './ui/shell/App.tsx';
import './ui/style.css';

const host = document.getElementById('root');
if (!host) throw new Error('missing #root');

createRoot(host).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
