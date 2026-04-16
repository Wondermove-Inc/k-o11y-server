import React, { useState, useCallback, useMemo, memo, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  RefreshState,
  AutoRefreshConfig,
  NetworkFilters
} from '../types';
import { openWindow, closeWindow, restart } from '../../../assets/ServiceMapIcons';
import './NetworkHeader.css';
import Typography from '../../../components/Typography/Typography';
import CustomTimeRange, { TimeRange, TimeRangeOption } from '../../../components/CustomTimeRange';

const autoRefreshIntervals = [
  { value: 10, label: '10s' },
  { value: 30, label: '30s' },
  { value: 60, label: '1m' },
  { value: 300, label: '5m' },
];

interface NetworkHeaderProps {
  timeRange: TimeRange;
  refreshState: RefreshState;
  onTimeRangeChange: (timeRange: TimeRange) => void;
  onManualRefresh: () => void;
  onAutoRefreshChange: (config: AutoRefreshConfig) => void;
  onQuickRefresh: () => void;
  isLoading: boolean;
  sidebarCollapsed: boolean;
  sidebarWidth?: number;
  onToggleSidebar: () => void;
  onPanelClose?: React.MutableRefObject<(() => void) | null>; // 패널 닫기 ref (선택적)
}

const NetworkHeader: React.FC<NetworkHeaderProps> = memo(({
  timeRange,
  refreshState,
  onTimeRangeChange,
  onManualRefresh,
  onAutoRefreshChange,
  onQuickRefresh,
  isLoading,
  sidebarCollapsed,
  sidebarWidth = 240,
  onToggleSidebar,
  onPanelClose,
}) => {
  const [isTimeRangePanelOpen, setIsTimeRangePanelOpen] = useState(false);
  const [isAutoRefreshDropdownOpen, setIsAutoRefreshDropdownOpen] = useState(false);
  const { t } = useTranslation('network_map');

  // Refs
  const autoRefreshDropdownRef = useRef<HTMLDivElement>(null);

  // 번역된 시간 범위 옵션
  const timeRangeOptions: TimeRangeOption[] = useMemo(() => [
    { value: 'last_5m', label: t('last_5_minutes'), minutes: 5 },
    { value: 'last_10m', label: t('last_10_minutes'), minutes: 10 },
    { value: 'last_15m', label: t('last_15_minutes'), minutes: 15 },
    { value: 'last_30m', label: t('last_30_minutes'), minutes: 30 },
    { value: 'last_1h', label: t('last_1_hour'), minutes: 60 },
    { value: 'last_3h', label: t('last_3_hours'), minutes: 180 },
    { value: 'last_24h', label: t('last_24_hours'), minutes: 1440 },
  ], [t]);

  // 외부 클릭 시 패널 닫기 기능
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      // 자동 새로고침 드롭다운 외부 클릭 감지
      if (autoRefreshDropdownRef.current && !autoRefreshDropdownRef.current.contains(event.target as Node)) {
        setIsAutoRefreshDropdownOpen(false);
      }
    };

    // 패널이 열려있을 때만 이벤트 리스너 추가
    if (isAutoRefreshDropdownOpen) {
      // mousedown과 click 이벤트 모두 감지 (React Flow 이벤트 전파 문제 해결)
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('click', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('click', handleClickOutside);
    };
  }, [isAutoRefreshDropdownOpen]);

  // Time Range Panel 토글
  const toggleTimeRangePanel = useCallback(() => {
    setIsTimeRangePanelOpen(prev => !prev);
  }, []);

  // 자동 새로고침 토글
  const toggleAutoRefresh = useCallback(() => {
    onAutoRefreshChange({
      ...refreshState.autoRefresh,
      enabled: !refreshState.autoRefresh.enabled,
    });
  }, [refreshState.autoRefresh, onAutoRefreshChange]);

  // 자동 새로고침 간격 변경
  const handleAutoRefreshIntervalChange = useCallback((intervalSeconds: number) => {
    onAutoRefreshChange({
      ...refreshState.autoRefresh,
      intervalSeconds,
    });

    setIsAutoRefreshDropdownOpen(false);
  }, [refreshState.autoRefresh, onAutoRefreshChange]);

  // 마지막 새로고침 시간 표시
  const formatLastRefreshTime = useCallback(() => {
    const now = new Date();
    const diffMs = now.getTime() - refreshState.lastRefreshed.getTime();
    const diffMinutes = Math.floor(diffMs / (1000 * 60));
    const diffSeconds = Math.floor(diffMs / 1000);

    if (diffMinutes > 0) {
      const unitKey = diffMinutes > 1 ? 'minutes' : 'minute';
      return `${diffMinutes} ${t(unitKey)} ${t('ago')}`;
    } else if (diffSeconds > 0) {
      const unitKey = diffSeconds > 1 ? 'seconds' : 'second';
      return `${diffSeconds} ${t(unitKey)} ${t('ago')}`;
    } else {
      return t('just_now');
    }
  }, [refreshState.lastRefreshed, t]);

  // 새로고침 핸들러
  const handleRefreshWithFilters = useCallback(async () => {
    onManualRefresh();
  }, [onManualRefresh]);

  // 현재 선택된 자동 새로고침 간격 라벨
  const currentAutoRefreshLabel = useMemo(() => {
    const option = autoRefreshIntervals.find(opt => opt.value === refreshState.autoRefresh.intervalSeconds);
    return option?.label || `${refreshState.autoRefresh.intervalSeconds}s`;
  }, [refreshState.autoRefresh.intervalSeconds]);

  return (
    <div className="network-header">
      {/* 로딩 표시 */}
      {isLoading && (
        <div className="header-loading">
          <span className="loading-indicator">⟳ Loading...</span>
        </div>
      )}

      <div
        className={`header-controls ${sidebarCollapsed ? 'no-border' : ''}`}
        style={!sidebarCollapsed ? { width: sidebarWidth + 1 } : undefined}
      >
        <div className="header-left">
          <div>
          <button
            className="filter-toggle-button"
            onClick={onToggleSidebar}
            aria-label={sidebarCollapsed ? t('expand_sidebar') : t('collapse_sidebar')}
            title={sidebarCollapsed ? t('open_filter_panel') : t('close_filter_panel')}
          >
            <img
              src={sidebarCollapsed ? closeWindow : openWindow}
              alt={sidebarCollapsed ? t('open') : t('close')}
              className="toggle-icon"
            />
            </button>
          </div>
          <div style={ {display: "flex"}}>
            <span className="header-filter-label">
              <Typography variant="b1" weight="regular">
                {t('filter')}
              </Typography>
            </span>
          </div>
          <div>
          <button
            className="filter-toggle-button"
            onClick={onQuickRefresh}
            aria-label={t('refresh_filters')}
            title={t('refresh_filters')}
            disabled={isLoading || refreshState.isRefreshing}
            >
            <img src={restart} alt={t('refresh')}/>
          </button>
          </div>
        </div>
      </div>

        <div className="header-right">
          {/* Time Range Picker */}
          <div className="control-group time-range-group">
            <CustomTimeRange
              timeRange={timeRange}
              onTimeRangeChange={onTimeRangeChange}
              isOpen={isTimeRangePanelOpen}
              onToggle={toggleTimeRangePanel}
              onPanelClose={onPanelClose}
              isLoading={isLoading}
              customTimeRangeOptions={timeRangeOptions}
            />
          </div>

          {/* 마지막 새로고침 시각 */}
          <div className="control-group last-updated-time-group">
          <label>{t('last_refresh_time')}</label>
           <span className="time-range-divider">|</span>
            <span className="last-updated-time-display">
              {formatLastRefreshTime()}
            </span>
          </div>

          {/* 자동 새로고침 설정 */}
          <div className="control-group auto-refresh-group">
            <label>{t('auto_refresh')}</label>
            <div className="auto-refresh-controls">
              <button
                className={`toggle-button ${refreshState.autoRefresh.enabled ? 'enabled' : 'disabled'}`}
                onClick={toggleAutoRefresh}
                disabled={isLoading}
                aria-pressed={refreshState.autoRefresh.enabled}
              >
                {refreshState.autoRefresh.enabled ? t('on') : t('off')}
              </button>

              {refreshState.autoRefresh.enabled && (
                <div className="dropdown-container interval-dropdown" ref={autoRefreshDropdownRef}>
                  <button
                    className={`dropdown-button ${isAutoRefreshDropdownOpen ? 'open' : ''}`}
                    onClick={() => setIsAutoRefreshDropdownOpen(!isAutoRefreshDropdownOpen)}
                    disabled={isLoading}
                    aria-expanded={isAutoRefreshDropdownOpen}
                    aria-haspopup="listbox"
                  >
                    {currentAutoRefreshLabel}
                  </button>
                  {isAutoRefreshDropdownOpen && (
                    <div className="dropdown-menu" role="listbox">
                      {autoRefreshIntervals.map(interval => (
                        <button
                          key={interval.value}
                          className={`dropdown-item ${refreshState.autoRefresh.intervalSeconds === interval.value ? 'selected' : ''}`}
                          onClick={() => handleAutoRefreshIntervalChange(interval.value)}
                          role="option"
                          aria-selected={refreshState.autoRefresh.intervalSeconds === interval.value}
                        >
                          {interval.label}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* 새로고침 버튼 */}
          <div className="control-group manual-refresh-group">
            <button
              className={`refresh-button ${refreshState.isRefreshing ? 'refreshing' : ''}`}
              onClick={handleRefreshWithFilters}
              disabled={isLoading || refreshState.isRefreshing}
              aria-label="새로고침"
              title="새로고침"
            >
              <span className={`refresh-icon ${refreshState.isRefreshing ? 'spinning' : ''}`}>
                <img src={restart} alt="restart"/>
              </span>
            </button>
          </div>
        </div>
    </div>
  );
});

// ✅ MUST: React.memo() displayName 설정
NetworkHeader.displayName = 'NetworkHeader';

export default NetworkHeader;
