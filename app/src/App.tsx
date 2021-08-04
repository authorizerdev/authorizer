import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { AuthorizerProvider } from '@authorizerdev/authorizer-react';
import Root from './Root';

export default function App() {
  // @ts-ignore
  const globalState: Record<string, string> = window['__authorizer__'];
  return (
    <div style={{ display: 'flex', justifyContent: 'center' }}>
      <div
        style={{
          width: 400,
          margin: `10px auto`,
          border: `1px solid #D1D5DB`,
          padding: `25px 20px`,
          borderRadius: 5,
        }}
      >
        <BrowserRouter>
          <AuthorizerProvider
            config={{
              authorizerURL: globalState.authorizerURL,
              redirectURL: globalState.redirectURL,
            }}
          >
            <Root />
          </AuthorizerProvider>
        </BrowserRouter>
      </div>
    </div>
  );
}
