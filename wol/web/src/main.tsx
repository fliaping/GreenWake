import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

async function prepare() {
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_API) {
    const { worker } = await import('./mocks/browser');
    await worker.start({
      onUnhandledRequest: 'bypass',
    });
  }
}

prepare().then(() => {
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
}); 