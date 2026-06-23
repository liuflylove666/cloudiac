import React, { useState, useContext, useMemo, memo, useEffect } from 'react';
import { Collapse } from 'antd';
import { useEventEmitter } from 'ahooks';
import { Eb_WP } from 'components/error-boundary';
import ResourceViewModal from 'components/resource-view-modal';
import GraphLayout from './graph-layout';
import TableLayout from './table-layout';
import DetailPageContext from '../../detail-page-context';

const { Panel } = Collapse;

const Resource = () => {

  const { type, sourceResourceId } = useContext(DetailPageContext);
  const event$ = useEventEmitter();
  const [ mode, setMode ] = useState(sourceResourceId ? 'table' : type === 'env' ? 'graph' : 'table');

  useEffect(() => {
    if (sourceResourceId) {
      setMode('table');
    }
  }, [sourceResourceId]);

  const content = useMemo(() => {
    const modeMap = {
      'graph': <GraphLayout setMode={setMode} />,
      'table': <TableLayout setMode={setMode} />
    }
    return modeMap[mode];
  });

  return (
    <>
      <Collapse expandIconPosition={'right'} defaultActiveKey={['1']} forceRender={true}>
        <Panel header='资源列表' key='1'>
          {content}
        </Panel>
      </Collapse>
      <ResourceViewModal event$={event$}/>
    </>
  );
};

export default Eb_WP()(memo(Resource));
