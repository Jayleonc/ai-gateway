import { createBrowserRouter } from 'react-router-dom';
import { App } from './App';
import { Dashboard } from './pages/Dashboard';
import { KeyDetail } from './pages/KeyDetail';

export const router = createBrowserRouter([
  {
    path: '/admin',
    element: <App />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
      {
        path: 'keys/:api_key_id',
        element: <KeyDetail />,
      },
    ],
  },
  {
    path: '/',
    element: <App />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
    ],
  },
]);
