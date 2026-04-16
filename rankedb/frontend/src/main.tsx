/**
 * @layer ui
 * @description Application entry point. Sets up MantineProvider with dark theme.
 * @depends ui/App
 * @must-not Contain business logic.
 */

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './ui/App';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
