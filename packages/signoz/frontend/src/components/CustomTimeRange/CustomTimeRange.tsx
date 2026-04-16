/**
 * CustomTimeRange Component
 *
 * 전역 Time Range Selector 컴포넌트
 * - Preset 시간 범위 선택 (Last 5m, 10m, 15m, 30m, 1h, 3h, 24h)
 * - Custom 시간 범위 입력 (Start/End DatePicker)
 * - Timezone 선택 및 변경
 *
 * @component
 */

import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import { useTimezone } from '../../providers/Timezone';
import { TIMEZONE_DATA } from '../CustomTimePicker/timezoneUtils';
import { whiteClock, calenderSvgBlue, searchSvg } from '../../assets/ServiceMapIcons';
import './CustomTimeRange.styles.scss';

export interface TimeRange {
  start: string; // ISO 8601 format
  end: string;   // ISO 8601 format
  preset?: string; // 'last_5m', 'last_10m', etc. or 'custom'
}

export interface TimeRangeOption {
  value: string;
  label: string;
  minutes: number;
}

interface CustomTimeRangeProps {
  /** 현재 선택된 시간 범위 */
  timeRange: TimeRange;
  /** 시간 범위 변경 핸들러 */
  onTimeRangeChange: (timeRange: TimeRange) => void;
  /** 패널 열림/닫힘 상태 */
  isOpen: boolean;
  /** 패널 열림/닫힘 토글 핸들러 */
  onToggle: () => void;
  /** 패널 닫기 ref (외부에서 닫기 요청) */
  onPanelClose?: React.MutableRefObject<(() => void) | null>;
  /** 로딩 상태 (optional) */
  isLoading?: boolean;
  /** 커스텀 시간 범위 옵션 (optional) */
  customTimeRangeOptions?: TimeRangeOption[];
}

const defaultTimeRangeOptions: TimeRangeOption[] = [
  { value: 'last_5m', label: 'Last 5 Minutes', minutes: 5 },
  { value: 'last_10m', label: 'Last 10 Minutes', minutes: 10 },
  { value: 'last_15m', label: 'Last 15 Minutes', minutes: 15 },
  { value: 'last_30m', label: 'Last 30 Minutes', minutes: 30 },
  { value: 'last_1h', label: 'Last 1 Hour', minutes: 60 },
  { value: 'last_3h', label: 'Last 3 Hours', minutes: 180 },
  { value: 'last_24h', label: 'Last 24 Hours', minutes: 1440 },
];

const CustomTimeRange: React.FC<CustomTimeRangeProps> = ({
  timeRange,
  onTimeRangeChange,
  isOpen,
  onToggle,
  onPanelClose,
  isLoading = false,
  customTimeRangeOptions,
}) => {
  const { t } = useTranslation('network_map');
  const { timezone, updateTimezone } = useTimezone();

  // 상태
  const [customStartTime, setCustomStartTime] = useState<Date | null>(null);
  const [customEndTime, setCustomEndTime] = useState<Date | null>(null);
  const [isTimezoneDropdownOpen, setIsTimezoneDropdownOpen] = useState(false);
  const [timezoneSearchTerm, setTimezoneSearchTerm] = useState('');

  // Refs
  const timeRangePanelRef = useRef<HTMLDivElement>(null);
  const timezoneDropdownRef = useRef<HTMLDivElement>(null);

  // 시간 범위 옵션 (customTimeRangeOptions가 있으면 사용, 없으면 기본값)
  const timeRangeOptions = useMemo(
    () => customTimeRangeOptions || defaultTimeRangeOptions,
    [customTimeRangeOptions],
  );

  // Timezone 필터링
  const filteredTimezones = useMemo(() => {
    if (!timezoneSearchTerm) return TIMEZONE_DATA;

    const normalizedSearch = timezoneSearchTerm.toLowerCase();
    return TIMEZONE_DATA.filter(
      (tz) =>
        tz.name.toLowerCase().includes(normalizedSearch) ||
        tz.offset.toLowerCase().includes(normalizedSearch) ||
        tz.searchIndex.toLowerCase().includes(normalizedSearch),
    );
  }, [timezoneSearchTerm]);

  // Custom 시간 범위 유효성 검증
  const isCustomTimeRangeValid = useMemo(() => {
    if (!customStartTime || !customEndTime) return false;
    return customStartTime < customEndTime;
  }, [customStartTime, customEndTime]);

  // 모든 패널 닫기
  const closeAllPanels = useCallback(() => {
    setIsTimezoneDropdownOpen(false);
  }, []);

  // 외부에서 패널 닫기 요청 처리
  useEffect(() => {
    if (onPanelClose) {
      onPanelClose.current = closeAllPanels;
    }
  }, [onPanelClose, closeAllPanels]);

  // 외부 클릭 시 패널 닫기
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      // Timezone 드롭다운 외부 클릭 감지
      if (
        timezoneDropdownRef.current &&
        !timezoneDropdownRef.current.contains(event.target as Node)
      ) {
        setIsTimezoneDropdownOpen(false);
      }
    };

    if (isTimezoneDropdownOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('click', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('click', handleClickOutside);
    };
  }, [isTimezoneDropdownOpen]);

  // 패널이 열릴 때 current time range를 custom 입력에 설정
  useEffect(() => {
    if (isOpen) {
      setCustomStartTime(new Date(timeRange.start));
      setCustomEndTime(new Date(timeRange.end));
    }
  }, [isOpen, timeRange.start, timeRange.end]);

  // 시간 범위 변경 핸들러
  const handleTimeRangeChange = useCallback(
    (preset: TimeRange['preset']) => {
      const option = timeRangeOptions.find((opt) => opt.value === preset);
      if (!option) return;

      const now = new Date();
      const start = new Date(now.getTime() - option.minutes * 60 * 1000);

      const newTimeRange = {
        start: start.toISOString(),
        end: now.toISOString(),
        preset: preset,
      };

      onTimeRangeChange(newTimeRange);
      onToggle(); // 패널 닫기
    },
    [onTimeRangeChange, timeRangeOptions, onToggle],
  );

  // Custom 시간 범위 적용
  const handleCustomTimeRangeApply = useCallback(() => {
    if (customStartTime && customEndTime && isCustomTimeRangeValid) {
      const customTimeRange: TimeRange = {
        start: customStartTime.toISOString(),
        end: customEndTime.toISOString(),
        preset: 'custom',
      };
      onTimeRangeChange(customTimeRange);
      onToggle(); // 패널 닫기
    }
  }, [customStartTime, customEndTime, isCustomTimeRangeValid, onTimeRangeChange, onToggle]);

  // 현재 시간 범위 표시
  const currentTimeRangeDisplay = useMemo(() => {
    if (timeRange.preset && timeRange.preset !== 'custom') {
      const option = timeRangeOptions.find((opt) => opt.value === timeRange.preset);
      return option?.label || t('last_15_minutes');
    } else {
      const start = new Date(timeRange.start).toLocaleTimeString('ko-KR', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      });
      const end = new Date(timeRange.end).toLocaleTimeString('ko-KR', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      });
      return `${start}-${end}`;
    }
  }, [timeRange, timeRangeOptions, t]);

  return (
    <div className="custom-time-range-container" ref={timeRangePanelRef}>
      {/* 버튼 */}
      <button
        className={`time-range-button ${isOpen ? 'open' : ''}`}
        onClick={onToggle}
        disabled={isLoading}
        aria-expanded={isOpen}
      >
        <span className="time-range-label">{t('time_range')}</span>
        <span className="time-range-divider">|</span>
        <div className="time-range-display">
          <img src={calenderSvgBlue} alt="calendar" className="calendar-icon" />
          <span className="time-range-text">{currentTimeRangeDisplay}</span>
        </div>
      </button>

      {/* 패널 - Portal로 렌더링 */}
      {isOpen && createPortal(
        <div className="time-range-panel" style={{
          position: 'fixed',
          top: timeRangePanelRef.current?.getBoundingClientRect().bottom ?? 0,
          left: timeRangePanelRef.current?.getBoundingClientRect().left ?? 0,
        }}>
          <div className="panel-content">
            {/* 왼쪽: Preset 옵션 */}
            <div className="left-section">
              <div className="preset-options">
                {timeRangeOptions.map((option) => (
                  <button
                    key={option.value}
                    className={`preset-option ${
                      timeRange.preset === option.value ? 'selected' : ''
                    }`}
                    onClick={() => handleTimeRangeChange(option.value)}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>

            <div className="section-divider"></div>

            {/* 오른쪽: Custom Time Range */}
            <div className="right-section">
              <div className="section-title">{t('custom_time_range')}</div>
              <div className="custom-inputs">
                {/* Start */}
                <div className="input-group">
                  <label>{t('start')}</label>
                  <div className="input-with-icon">
                    <DatePicker
                      selected={customStartTime}
                      onChange={(date: Date | null) => setCustomStartTime(date)}
                      showTimeSelect
                      timeFormat="HH:mm"
                      timeIntervals={15}
                      dateFormat="yyyy. MM. dd. HH:mm"
                      timeCaption={t('time_label')}
                      placeholderText={t('start')}
                      className="time-input"
                      popperClassName="custom-datepicker-popper"
                      calendarClassName="custom-datepicker-calendar"
                      showPopperArrow={false}
                      timeInputLabel={`${t('time_label')}:`}
                    />
                    <img
                      src={whiteClock}
                      alt="clock"
                      className="clock-icon"
                      onClick={() => {
                        const input = document.querySelector(
                          '.input-with-icon .react-datepicker__input-container input',
                        ) as HTMLInputElement;
                        if (input) {
                          input.focus();
                        }
                      }}
                    />
                  </div>
                </div>

                {/* End */}
                <div className="input-group">
                  <label>{t('end')}</label>
                  <div className="input-with-icon">
                    <DatePicker
                      selected={customEndTime}
                      onChange={(date: Date | null) => setCustomEndTime(date)}
                      showTimeSelect
                      timeFormat="HH:mm"
                      timeIntervals={15}
                      dateFormat="yyyy. MM. dd. HH:mm"
                      timeCaption={t('time_label')}
                      placeholderText={t('end')}
                      className="time-input"
                      popperClassName="custom-datepicker-popper"
                      calendarClassName="custom-datepicker-calendar"
                      showPopperArrow={false}
                      timeInputLabel={`${t('time_label')}:`}
                    />
                    <img
                      src={whiteClock}
                      alt="clock"
                      className="clock-icon"
                      onClick={() => {
                        const inputs = document.querySelectorAll(
                          '.input-with-icon .react-datepicker__input-container input',
                        ) as NodeListOf<HTMLInputElement>;
                        if (inputs[1]) {
                          inputs[1].focus();
                        }
                      }}
                    />
                  </div>
                </div>
              </div>

              {/* 유효성 검증 에러 */}
              {customStartTime && customEndTime && !isCustomTimeRangeValid && (
                <div className="validation-error">{t('invalid_time_order')}</div>
              )}

              {/* Apply 버튼 */}
              <button
                className="apply-button"
                onClick={handleCustomTimeRangeApply}
                disabled={!isCustomTimeRangeValid}
              >
                {t('apply')}
              </button>
            </div>
          </div>

          {/* Timezone 섹션 */}
          <div className="timezone-section" ref={timezoneDropdownRef}>
            <div className="timezone-label">{t('Current Timezone') || '현재 시간대'}</div>
            <button
              className={`timezone-selector-button ${isTimezoneDropdownOpen ? 'open' : ''}`}
              type="button"
              onClick={() => setIsTimezoneDropdownOpen(!isTimezoneDropdownOpen)}
              aria-expanded={isTimezoneDropdownOpen}
            >
              <span className="timezone-display">
                ({timezone.offset}) {timezone.name}
              </span>
              <span className="timezone-dropdown-icon">▾</span>
            </button>

            {/* Timezone 드롭다운 */}
            {isTimezoneDropdownOpen && (
              <div className="timezone-dropdown">
                <div className="timezone-search">
                  <img
                    className="timezone-search-icon"
                    src={searchSvg}
                    alt=""
                    aria-hidden="true"
                  />
                  <input
                    type="text"
                    className="timezone-search-input"
                    placeholder={t('Search Timezones...') || '시간대 검색'}
                    value={timezoneSearchTerm}
                    onChange={(e) => setTimezoneSearchTerm(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Escape') {
                        setIsTimezoneDropdownOpen(false);
                        setTimezoneSearchTerm('');
                      }
                    }}
                    autoFocus
                  />
                </div>
                <div className="timezone-list">
                  {filteredTimezones.map((tz) => (
                    <button
                      key={tz.value}
                      className={`timezone-item ${timezone.value === tz.value ? 'selected' : ''}`}
                      type="button"
                      onClick={() => {
                        updateTimezone(tz);
                        setIsTimezoneDropdownOpen(false);
                        setTimezoneSearchTerm('');
                      }}
                    >
                      <span className="timezone-item-check">
                        {timezone.value === tz.value && '✓'}
                      </span>
                      <span className="timezone-item-name">
                        ({tz.offset}) {tz.name}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>,
        document.body
      )}
    </div>
  );
};

export default CustomTimeRange;
