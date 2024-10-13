import React, {
  FC,
  createContext,
  useReducer,
  useContext,
  useRef,
  useEffect,
} from 'react';
import { Authorizer, User, AuthToken } from '@authorizerdev/authorizer-js';

import {
  AuthorizerContextPropsType,
  AuthorizerState,
  AuthorizerProviderAction,
} from '../types';
import { AuthorizerProviderActionType } from '../constants';
import { hasWindow } from '../utils/window';

const AuthorizerContext = createContext<AuthorizerContextPropsType>({
  config: {
    authorizerURL: '',
    redirectURL: '/',
    client_id: '',
    is_google_login_enabled: false,
    is_github_login_enabled: false,
    is_facebook_login_enabled: false,
    is_linkedin_login_enabled: false,
    is_apple_login_enabled: false,
    is_twitter_login_enabled: false,
    is_microsoft_login_enabled: false,
    is_twitch_login_enabled: false,
    is_roblox_login_enabled: false,
    is_email_verification_enabled: false,
    is_basic_authentication_enabled: false,
    is_magic_link_login_enabled: false,
    is_sign_up_enabled: false,
    is_strong_password_enabled: true,
    is_multi_factor_auth_enabled: false,
    is_mobile_basic_authentication_enabled: false,
    is_phone_verification_enabled: false,
  },
  user: null,
  token: null,
  loading: false,
  setLoading: () => {},
  setToken: () => {},
  setUser: () => {},
  setAuthData: () => {},
  authorizerRef: new Authorizer({
    authorizerURL: `http://localhost:8080`,
    redirectURL: hasWindow() ? window.location.origin : '/',
    clientID: '',
  }),
  logout: async () => {},
});

function reducer(
  state: AuthorizerState,
  action: AuthorizerProviderAction
): AuthorizerState {
  switch (action.type) {
    case AuthorizerProviderActionType.SET_USER:
      return { ...state, user: action.payload.user };
    case AuthorizerProviderActionType.SET_TOKEN:
      return {
        ...state,
        token: action.payload.token,
      };
    case AuthorizerProviderActionType.SET_LOADING:
      return {
        ...state,
        loading: action.payload.loading,
      };
    case AuthorizerProviderActionType.SET_CONFIG:
      return {
        ...state,
        config: action.payload.config,
      };
    case AuthorizerProviderActionType.SET_AUTH_DATA:
      return {
        ...action.payload,
      };

    default:
      throw new Error();
  }
}

let initialState: AuthorizerState = {
  user: null,
  token: null,
  loading: true,
  config: {
    authorizerURL: '',
    redirectURL: '/',
    client_id: '',
    is_google_login_enabled: false,
    is_github_login_enabled: false,
    is_facebook_login_enabled: false,
    is_linkedin_login_enabled: false,
    is_apple_login_enabled: false,
    is_twitter_login_enabled: false,
    is_microsoft_login_enabled: false,
    is_twitch_login_enabled: false,
    is_roblox_login_enabled: false,
    is_email_verification_enabled: false,
    is_basic_authentication_enabled: false,
    is_magic_link_login_enabled: false,
    is_sign_up_enabled: false,
    is_strong_password_enabled: true,
    is_multi_factor_auth_enabled: false,
    is_mobile_basic_authentication_enabled: false,
    is_phone_verification_enabled: false,
  },
};

export const AuthorizerProvider: FC<{
  children: React.ReactNode;
  config: {
    authorizerURL: string;
    redirectURL: string;
    clientID?: string;
  };
  onStateChangeCallback?: (stateData: AuthorizerState) => Promise<void>;
}> = ({ config: defaultConfig, onStateChangeCallback, children }) => {
  const [state, dispatch] = useReducer(reducer, {
    ...initialState,
    config: {
      ...initialState.config,
      ...defaultConfig,
    },
  });

  let intervalRef: any = null;

  const authorizerRef = useRef(
    new Authorizer({
      authorizerURL: state.config.authorizerURL,
      redirectURL: hasWindow()
        ? state.config.redirectURL || window.location.origin
        : state.config.redirectURL || '/',
      clientID: state.config.client_id,
    })
  );

  const getToken = async () => {
    const {
      data: metaRes,
      errors: metaResErrors,
    } = await authorizerRef.current.getMetaData();
    try {
      if (metaResErrors && metaResErrors.length) {
        throw new Error(metaResErrors[0].message);
      }
      const { data: res, errors } = await authorizerRef.current.getSession();
      if (errors && errors.length) {
        throw new Error(errors[0].message);
      }
      if (res && res.access_token && res.user) {
        const token = {
          access_token: res.access_token,
          expires_in: res.expires_in,
          id_token: res.id_token,
          refresh_token: res.refresh_token || '',
        };
        dispatch({
          type: AuthorizerProviderActionType.SET_AUTH_DATA,
          payload: {
            ...state,
            token,
            user: res.user,
            config: {
              ...state.config,
              ...metaRes,
            },
            loading: false,
          },
        });

        // const millisecond = getIntervalDiff(res.expires_at);
        // if (millisecond > 0) {
        //   if (intervalRef) clearInterval(intervalRef);
        //   intervalRef = setInterval(() => {
        //     getToken();
        //   }, millisecond);
        // }
        if (intervalRef) clearInterval(intervalRef);
        intervalRef = setInterval(() => {
          getToken();
        }, res.expires_in * 1000);
      } else {
        dispatch({
          type: AuthorizerProviderActionType.SET_AUTH_DATA,
          payload: {
            ...state,
            token: null,
            user: null,
            config: {
              ...state.config,
              ...metaRes,
            },
            loading: false,
          },
        });
      }
    } catch (err) {
      dispatch({
        type: AuthorizerProviderActionType.SET_AUTH_DATA,
        payload: {
          ...state,
          token: null,
          user: null,
          config: {
            ...state.config,
            ...metaRes,
          },
          loading: false,
        },
      });
    }
  };

  useEffect(() => {
    getToken();
    return () => {
      if (intervalRef) {
        clearInterval(intervalRef);
      }
    };
  }, []);

  useEffect(() => {
    if (onStateChangeCallback) {
      onStateChangeCallback(state);
    }
  }, [state]);

  const handleTokenChange = (token: AuthToken | null) => {
    dispatch({
      type: AuthorizerProviderActionType.SET_TOKEN,
      payload: {
        token,
      },
    });

    if (token?.access_token) {
      if (intervalRef) clearInterval(intervalRef);
      intervalRef = setInterval(() => {
        getToken();
      }, token.expires_in * 1000);
    }
  };

  const setAuthData = (data: AuthorizerState) => {
    dispatch({
      type: AuthorizerProviderActionType.SET_AUTH_DATA,
      payload: data,
    });

    if (data.token?.access_token) {
      if (intervalRef) clearInterval(intervalRef);
      intervalRef = setInterval(() => {
        getToken();
      }, data.token.expires_in * 1000);
    }
  };

  const setUser = (user: User | null) => {
    dispatch({
      type: AuthorizerProviderActionType.SET_USER,
      payload: {
        user,
      },
    });
  };

  const setLoading = (loading: boolean) => {
    dispatch({
      type: AuthorizerProviderActionType.SET_LOADING,
      payload: {
        loading,
      },
    });
  };

  const logout = async () => {
    dispatch({
      type: AuthorizerProviderActionType.SET_LOADING,
      payload: {
        loading: true,
      },
    });
    await authorizerRef.current.logout();
    const loggedOutState = {
      user: null,
      token: null,
      loading: false,
      config: state.config,
    };
    dispatch({
      type: AuthorizerProviderActionType.SET_AUTH_DATA,
      payload: loggedOutState,
    });
  };

  return (
    <AuthorizerContext.Provider
      value={{
        ...state,
        setUser,
        setLoading,
        setToken: handleTokenChange,
        setAuthData: setAuthData,
        authorizerRef: authorizerRef.current,
        logout,
      }}
    >
      {children}
    </AuthorizerContext.Provider>
  );
};

export const useAuthorizer = () => useContext(AuthorizerContext);
