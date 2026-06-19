import React from 'react';
import { createRoot } from 'react-dom/client';
import PropTypes from 'prop-types';
import { Provider } from 'react-redux';
import { ConnectedRouter } from 'connected-react-router';
import history from 'utils/history';
import configureStore from './configureStore';
import App from 'containers/App';
import LanguageProvider from 'containers/LanguageProvider';
import { translationMessages } from './i18n';
import './publicPath';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/es/locale/zh_CN';
import moment from 'moment';
import 'moment/locale/zh-cn';
import 'containers/App/styles/index.less';
moment.locale('zh-cn');

const initialState = {};
const store = configureStore(initialState, history);
const MOUNT_NODE = document.getElementById('app');
let root;

const ConnectedApp = props => (
  <Provider store={store}>
    <LanguageProvider messages={props.messages}>
      <ConfigProvider locale={zhCN}>
        <ConnectedRouter history={history}>
          <App />
        </ConnectedRouter>
      </ConfigProvider>
    </LanguageProvider>
  </Provider>
);

ConnectedApp.propTypes = {
  messages: PropTypes.object
};

const render = messages => {
  if (!root) {
    root = createRoot(MOUNT_NODE);
  }
  root.render(<ConnectedApp messages={messages} />);
};

const unmountRoot = () => {
  if (root) {
    root.unmount();
    root = null;
  }
};

if (module.hot) {
  // Hot reloadable React components and translation json files
  // modules.hot.accept does not accept dynamic dependencies,
  // have to be constants at compile-time
  module.hot.accept(['./i18n'], () => {
    unmountRoot();
    render(translationMessages);
  });
}


if (!window.__POWERED_BY_QIANKUN__) { // do sth not in qiankun
  render(translationMessages);
}

export async function bootstrap() { // do sth in qiankun
  // console.log('[boilerplate] react app bootstraped');
}

export async function mount(props) {
  // console.log('[boilerplate] props from main framework', props);
  render(translationMessages);
}

export async function unmount() {
  unmountRoot();
}
