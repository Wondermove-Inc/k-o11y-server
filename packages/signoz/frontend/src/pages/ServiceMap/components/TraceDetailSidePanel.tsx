import React, { memo, useEffect, useMemo } from 'react';
import { generatePath } from 'react-router-dom';
import { closeWindow } from '../../../assets/ServiceMapIcons';
import './TraceDetailSidePanel.css';

interface TraceDetailSidePanelProps {
  traceId: string;
  onClose: () => void;
}

const TraceDetailSidePanel: React.FC<TraceDetailSidePanelProps> = memo(({ traceId, onClose }) => {
	const tracePath = useMemo(
		() => generatePath('/trace-embed/:id', { id: traceId }),
		[traceId],
	);

	useEffect(() => {
		const handleMessage = (event: MessageEvent): void => {
			if (event.origin !== window.location.origin) {
				return;
			}

			if (event.data?.type === 'KO11Y_TRACE_EMBED_CLOSE') {
				onClose();
			}
		};

		window.addEventListener('message', handleMessage);

		return () => {
			window.removeEventListener('message', handleMessage);
		};
	}, [onClose]);

  return (
    <div className="trace-detail-side-panel">
      <div className="trace-detail-side-panel-header">
        <div className="trace-detail-side-panel-title-wrap">
          <h3>Trace Details</h3>
          <div className="trace-detail-side-panel-subtitle" title={traceId}>
            {traceId}
          </div>
        </div>
        <button type="button" className="trace-detail-side-panel-close" onClick={onClose} aria-label="Close trace details panel">
          <img src={closeWindow} alt="close" />
        </button>
      </div>

      <div className="trace-detail-side-panel-content">
        <iframe title={`trace-${traceId}`} src={tracePath} className="trace-detail-side-panel-iframe" />
      </div>
    </div>
  );
});

TraceDetailSidePanel.displayName = 'TraceDetailSidePanel';

export default TraceDetailSidePanel;
