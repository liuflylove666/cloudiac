import React, { useState, useCallback, useEffect } from 'react';
import { Spin, Tabs } from 'antd';
import { connect } from 'react-redux';
import { compose } from 'redux';

import PageHeader from 'components/pageHeader';
import { Eb_WP } from 'components/error-boundary';
import Layout from 'components/common/layout';


import Basic from './pages/basic';
import Pwd from './pages/pwd';

const subNavs = {
  basic: '基本信息',
  pwd: '修改密码'
};

const User = ({ userInfo, dispatch }) => {

  const [ panel, setPanel ] = useState('basic');
  const hasUserInfo = !!(userInfo && (userInfo.id || userInfo.email));

  useEffect(() => {
    if (!hasUserInfo) {
      dispatch({
        type: 'global/getUserInfo'
      });
    }
  }, [ hasUserInfo ]);

  const updateUserInfo = ({ payload, cb }) => {
    dispatch({
      type: 'global/updateUserInfo',
      payload: {
        ...payload,
        id: userInfo.id
      },
      cb
    });
  };

  const renderByPanel = useCallback(() => {
    if (!hasUserInfo) {
      return (
        <div style={{ minHeight: 220, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Spin tip='加载用户信息...' />
        </div>
      );
    }
    const PAGES = {
      basic: Basic,
      pwd: Pwd
    };
    const Page = PAGES[panel] || Basic;
    return <Page
      title={subNavs[panel]}
      userInfo={userInfo}
      updateUserInfo={updateUserInfo}
    />;
  }, [ panel, userInfo, hasUserInfo ]);

  return <Layout
    extraHeader={<PageHeader
      className='container-inner-width'
      title='用户设置'
      showIcon={'user'}
      breadcrumb={false}
      renderFooter={() => (
        <Tabs
          tabBarStyle={{ backgroundColor: '#fff' }}
          animated={false}
          renderTabBar={(props, DefaultTabBar) => {
            return (
              <div style={{ marginBottom: -16 }}>
                <DefaultTabBar {...props} />
              </div>
            );
          }}
          activeKey={panel}
          onChange={(k) => setPanel(k)}
        >
          {Object.keys(subNavs).map((it) => (
            <Tabs.TabPane
              tab={subNavs[it]}
              key={it}
            />
          ))}
        </Tabs>
      )}
    />}
  >
    <div className='idcos-card container-inner-width'>
      {renderByPanel()}
    </div>
  </Layout>;
};

const mapStateToProps = (state) => {
  return {
    userInfo: state.global.get('userInfo').toJS()
  };
};

const withConnect = connect(
  mapStateToProps
);
const withEB = Eb_WP();

export default compose(
  withConnect,
  withEB
)(User);
