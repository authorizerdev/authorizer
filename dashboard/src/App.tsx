import * as React from "react";
import {   ChakraProvider, extendTheme } from "@chakra-ui/react";
import { BrowserRouter } from "react-router-dom";
import { createClient, Provider } from "urql";
import {AppRoutes} from './routes'
import { AuthContainer } from "./containers/AuthContainer";

const queryClient = createClient({
  url: "/graphql",
  fetchOptions: () => {
    return {
      credentials: "include",
    };
  },
});

const theme = extendTheme({
  styles: {
    global: {
      "html, body, #root": {
        height: "100%",
      },
    },
  },
  colors: {
    blue: {
      500: "rgb(59,130,246)",
    },
  },
});

export default function App() {
  return (
    <ChakraProvider theme={theme}>
      <Provider value={queryClient}>
        <BrowserRouter basename="/dashboard">
          <AuthContainer>
            <AppRoutes />
          </AuthContainer>
        </BrowserRouter>
      </Provider>
    </ChakraProvider>
  );
}
