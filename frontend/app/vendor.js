import * as React from 'react';
import * as ReactDOMModule from 'react-dom';
import * as ReactDOMClient from 'react-dom/client';
import * as ReactRedux from 'react-redux';
import * as ReactRouterDOM from 'react-router-dom';
import * as ConnectedReactRouter from 'connected-react-router';
import * as ReduxSaga from 'redux-saga';
import * as Redux from 'redux';
import * as lodash from 'lodash';
import * as ReactIntl from 'react-intl';
import * as Reselect from 'reselect';
import * as moment from 'moment';

const legacyRoots = new WeakMap();

const getNodeFromFiber = fiber => {
  let next = fiber;

  while (next) {
    const stateNode = next.stateNode;
    if (stateNode && typeof stateNode === 'object' && stateNode.nodeType) {
      return stateNode;
    }

    if (next.child) {
      next = next.child;
      continue;
    }

    while (next && !next.sibling) {
      next = next.return;
    }

    next = next && next.sibling;
  }

  return null;
};

const ReactDOM = {
  ...ReactDOMModule,
  findDOMNode(componentOrElement) {
    if (!componentOrElement) {
      return null;
    }

    if (componentOrElement.nodeType) {
      return componentOrElement;
    }

    if (componentOrElement.nativeElement && componentOrElement.nativeElement.nodeType) {
      return componentOrElement.nativeElement;
    }

    if (typeof componentOrElement.getDOMNode === 'function') {
      return componentOrElement.getDOMNode();
    }

    return getNodeFromFiber(
      componentOrElement._reactInternals || componentOrElement._reactInternalFiber
    );
  },
  render(element, container, callback) {
    let legacyInstance;
    let nextElement = element;

    if (React.isValidElement(element) && typeof callback === 'function') {
      const originalRef = element.ref || (element.props && element.props.ref);

      nextElement = React.cloneElement(element, {
        ref(instance) {
          legacyInstance = instance;

          if (typeof originalRef === 'function') {
            originalRef(instance);
          } else if (originalRef && typeof originalRef === 'object') {
            originalRef.current = instance;
          }
        }
      });
    }

    let root = legacyRoots.get(container);
    if (!root) {
      root = ReactDOMClient.createRoot(container);
      legacyRoots.set(container, root);
    }

    const render = () => root.render(nextElement);
    if (typeof ReactDOMModule.flushSync === 'function') {
      ReactDOMModule.flushSync(render);
    } else {
      render();
    }

    if (typeof callback === 'function') {
      callback.call(legacyInstance);
    }

    return legacyInstance;
  },
  unmountComponentAtNode(container) {
    const root = legacyRoots.get(container);

    if (!root) {
      return false;
    }

    root.unmount();
    legacyRoots.delete(container);
    return true;
  }
};

const libs = {
  React,
  ReactDOM,
  ReactDOMClient,
  ReactRedux,
  ReactRouterDOM,
  ConnectedReactRouter,
  ReduxSaga,
  Redux,
  lodash,
  ReactIntl,
  Reselect,
  moment
};

Object.keys(libs).forEach(key => {
  window[key] = libs[key];
});
