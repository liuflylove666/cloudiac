import React, { useEffect } from 'react';
import RoutesList from 'components/routes-list';
import routes from 'routes';
import reducer from './reducer';
import saga from './saga';
import { useInjectReducer, useInjectSaga } from "redux-injectors";
import { useDispatch } from 'react-redux';

import { withRouter } from 'react-router-dom';

import AppHeader from 'components/AppHeader';

import queryString from 'query-string';

const KEY = 'global';

function App(props) {
  const { location } = props;
  const { pathname, search } = location;
  const dispatch = useDispatch();
  useInjectReducer({ key: KEY, reducer });
  useInjectSaga({ key: KEY, saga });
  
  useEffect(() => {
    if (freeLoginCheck()) {
      return;
    }
    if (!window.__POWERED_BY_QIANKUN__) {
      dispatch({
        type: 'global/getOrgs',
        payload: {
          status: 'enable'
        }
      });
      dispatch({
        type: 'global/getUserInfo'
      });
    }
  }, []);

  // 免登检查
  const freeLoginCheck = () => {
    const { token } = queryString.parse(search);
    if (token) {
      localStorage.accessToken = token;
      window.location.search = '';
      return true;
    }
    return false;
  };

  return (
    <div className='idcos-app'>
      <AppHeader
        theme='dark'
        locationPathName={pathname}
      />
      <RoutesList routes={routes()}/>
    </div>
  );
}

export default withRouter(App);
