/**
 * TimeRangeSelector Component
 *
 * TopNav용 Time Range Selector (ServiceMap CustomTimeRange와 동일한 UI)
 * - ServiceMap의 CustomTimeRange 디자인 적용
 * - Preset 옵션, Custom DatePicker (react-datepicker), Timezone 선택
 *
 * @component
 */

import './TimeRangeSelector.styles.scss';

import { Button } from 'antd';
import logEvent from 'api/common/logEvent';
import cx from 'classnames';
import TimezonePicker from 'components/CustomTimePicker/TimezonePicker';
import { TIMEZONE_DATA } from 'components/CustomTimePicker/timezoneUtils';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { DateTimeRangeType } from 'container/TopNav/CustomDateTimeModal';
import {
	LexicalContext,
} from 'container/TopNav/DateTimeSelectionV2/config';
import { useTimezone } from 'providers/Timezone';
import {
	Dispatch,
	SetStateAction,
	useCallback,
	useMemo,
	useState,
	useEffect,
	useRef,
	forwardRef,
} from 'react';
import { useLocation } from 'react-router-dom';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import { whiteClock, searchSvg } from '../../assets/ServiceMapIcons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';

interface TimeRangeSelectorProps {
	options: any[];
	setIsOpen: Dispatch<SetStateAction<boolean>>;
	customDateTimeVisible: boolean;
	setCustomDTPickerVisible: Dispatch<SetStateAction<boolean>>;
	onCustomDateHandler: (
		dateTimeRange: DateTimeRangeType,
		lexicalContext?: LexicalContext,
	) => void;
	onSelectHandler: (label: string, value: string) => void;
	onGoLive: () => void;
	selectedTime: string;
	activeView: 'datetime' | 'timezone';
	setActiveView: Dispatch<SetStateAction<'datetime' | 'timezone'>>;
	isOpenedFromFooter: boolean;
	setIsOpenedFromFooter: Dispatch<SetStateAction<boolean>>;
	onExitLiveLogs: () => void;
	minTime: number;
	maxTime: number;
}

// DatePicker의 customInput을 위한 컴포넌트
const CustomInput = forwardRef<
	HTMLInputElement,
	{ value?: string; onClick?: () => void; placeholder?: string }
>(({ value, onClick, placeholder }, ref) => (
	<div className="trs-custom-input-wrapper" onClick={onClick}>
		<input
			ref={ref}
			className="trs-time-input"
			value={value}
			readOnly
			placeholder={placeholder}
		/>
		<img src={whiteClock} alt="clock" className="trs-clock-icon" />
	</div>
));

CustomInput.displayName = 'CustomInput';

function TimeRangeSelector({
	options,
	setIsOpen,
	customDateTimeVisible,
	setCustomDTPickerVisible,
	onCustomDateHandler,
	onSelectHandler,
	onGoLive,
	selectedTime,
	activeView,
	setActiveView,
	isOpenedFromFooter,
	setIsOpenedFromFooter,
	onExitLiveLogs,
	minTime,
	maxTime,
}: TimeRangeSelectorProps): JSX.Element {
	const { pathname } = useLocation();
	const { t } = useTranslation('network_map');

	const isLogsExplorerPage = useMemo(() => pathname === ROUTES.LOGS_EXPLORER, [
		pathname,
	]);

	const url = new URLSearchParams(window.location.search);

	let panelTypeFromURL = url.get(QueryParams.panelTypes);

	try {
		panelTypeFromURL = JSON.parse(panelTypeFromURL as string);
	} catch {
		// fallback → leave as-is
	}

	const isLogsListView =
		panelTypeFromURL !== 'table' && panelTypeFromURL !== 'graph';

	const { timezone, updateTimezone } = useTimezone();

	// selectedTime이 'custom'이면 현재 설정된 시간을 사용, 아니면 기본값(15분 전 ~ 현재)
	const [customStartTime, setCustomStartTime] = useState<Date | null>(() => {
		if (selectedTime === 'custom' && minTime) {
			return new Date(minTime / 1000000); // nanoseconds to milliseconds
		}
		return new Date(Date.now() - 15 * 60 * 1000);
	});

	const [customEndTime, setCustomEndTime] = useState<Date | null>(() => {
		if (selectedTime === 'custom' && maxTime) {
			return new Date(maxTime / 1000000); // nanoseconds to milliseconds
		}
		return new Date();
	});
	const [isTimezoneDropdownOpen, setIsTimezoneDropdownOpen] = useState(false);
	const [timezoneSearchTerm, setTimezoneSearchTerm] = useState('');

	const timezoneDropdownRef = useRef<HTMLDivElement>(null);

	const handleExitLiveLogs = useCallback((): void => {
		if (isLogsExplorerPage) {
			onExitLiveLogs();
		}
	}, [isLogsExplorerPage, onExitLiveLogs]);

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

	const isCustomTimeRangeValid = useMemo(() => {
		if (!customStartTime || !customEndTime) return false;
		return customStartTime < customEndTime;
	}, [customStartTime, customEndTime]);

	const handleCustomTimeRangeApply = useCallback(() => {
		if (customStartTime && customEndTime && isCustomTimeRangeValid) {
			handleExitLiveLogs();
			onCustomDateHandler([dayjs(customStartTime), dayjs(customEndTime)]);
			setIsOpen(false);
		}
	}, [customStartTime, customEndTime, isCustomTimeRangeValid, onCustomDateHandler, setIsOpen, handleExitLiveLogs]);

	useEffect(() => {
		const handleClickOutside = (event: MouseEvent) => {
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

	const handleTimezoneHintClick = (): void => {
		setActiveView('timezone');
		setIsOpenedFromFooter(true);
		logEvent(
			'DateTimePicker: Timezone picker opened from time range picker footer',
			{
				page: pathname,
			},
		);
	};

	if (activeView === 'timezone') {
		return (
			<div className="time-range-selector-popover">
				<TimezonePicker
					setActiveView={setActiveView}
					setIsOpen={setIsOpen}
					isOpenedFromFooter={isOpenedFromFooter}
				/>
			</div>
		);
	}

	const handleGoLive = (): void => {
		onGoLive();
		setIsOpen(false);
	};

	return (
		<div className="time-range-selector-popover">
			<div className="trs-panel-content">
				{/* 왼쪽: Preset 옵션 */}
				<div className="trs-left-section">
					<div className="trs-preset-options">
						{isLogsExplorerPage && isLogsListView && (
							<Button
								className="trs-preset-option trs-live-option"
								type="text"
								onClick={handleGoLive}
							>
								Live
							</Button>
						)}
						{options
							.filter((option) => option.label !== 'Custom' && option.label !== 'Last 1 month')
							.map((option) => (
								<button
									key={option.value}
									className={cx(
										'trs-preset-option',
										selectedTime === option.value && 'selected',
									)}
									onClick={(): void => {
										handleExitLiveLogs();
										onSelectHandler(option.label, option.value);
									}}
								>
									{option.label}
								</button>
							))}
					</div>
				</div>

				<div className="trs-section-divider"></div>

				{/* 오른쪽: Custom Time Range (ServiceMap 스타일) */}
				<div className="trs-right-section">
					<div className="trs-section-title">{t('custom_time_range') || 'Custom Time Range'}</div>
					<div className="trs-custom-inputs">
						<div className="trs-input-group">
							<label>{t('start') || 'Start'}</label>
							<DatePicker
								selected={customStartTime}
								onChange={(date: Date | null) => setCustomStartTime(date)}
								showTimeSelect
								timeFormat="HH:mm"
								timeIntervals={15}
								dateFormat="yyyy. MM. dd. HH:mm"
								timeCaption={t('time_label') || 'Time'}
								placeholderText={t('start') || 'Start'}
								popperClassName="custom-datepicker-popper"
								calendarClassName="custom-datepicker-calendar"
								showPopperArrow={false}
								timeInputLabel={`${t('time_label') || 'Time'}:`}
								customInput={<CustomInput />}
							/>
						</div>
						<div className="trs-input-group">
							<label>{t('end') || 'End'}</label>
							<DatePicker
								selected={customEndTime}
								onChange={(date: Date | null) => setCustomEndTime(date)}
								showTimeSelect
								timeFormat="HH:mm"
								timeIntervals={15}
								dateFormat="yyyy. MM. dd. HH:mm"
								timeCaption={t('time_label') || 'Time'}
								placeholderText={t('end') || 'End'}
								popperClassName="custom-datepicker-popper"
								calendarClassName="custom-datepicker-calendar"
								showPopperArrow={false}
								timeInputLabel={`${t('time_label') || 'Time'}:`}
								customInput={<CustomInput />}
							/>
						</div>
					</div>

					{/* 유효성 검증 에러 */}
					{customStartTime && customEndTime && !isCustomTimeRangeValid && (
						<div className="trs-validation-error">
							{t('invalid_time_order') || 'Start time must be before end time'}
						</div>
					)}

					{/* Apply 버튼 */}
					<button
						className="trs-apply-button"
						onClick={handleCustomTimeRangeApply}
						disabled={!isCustomTimeRangeValid}
					>
						{t('apply') || 'Apply'}
					</button>
				</div>
			</div>

			{/* 하단: Timezone 섹션 (ServiceMap 스타일 - 드롭다운 방식) */}
			<div className="trs-timezone-section" ref={timezoneDropdownRef}>
				<div className="trs-timezone-label">{t('Current Timezone') || '현재 시간대'}</div>
				<button
					className={`trs-timezone-selector-button ${isTimezoneDropdownOpen ? 'open' : ''}`}
					type="button"
					onClick={() => setIsTimezoneDropdownOpen(!isTimezoneDropdownOpen)}
					aria-expanded={isTimezoneDropdownOpen}
				>
					<span className="trs-timezone-display">
						({timezone.offset}) {timezone.name}
					</span>
					<span className="trs-timezone-dropdown-icon">▾</span>
				</button>

				{/* Timezone 드롭다운 */}
				{isTimezoneDropdownOpen && (
					<div className="trs-timezone-dropdown">
						<div className="trs-timezone-search">
							<img
								className="trs-timezone-search-icon"
								src={searchSvg}
								alt=""
								aria-hidden="true"
							/>
							<input
								type="text"
								className="trs-timezone-search-input"
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
						<div className="trs-timezone-list">
							{filteredTimezones.map((tz) => (
								<button
									key={tz.value}
									className={`trs-timezone-item ${timezone.value === tz.value ? 'selected' : ''}`}
									type="button"
									onClick={() => {
										updateTimezone(tz);
										setIsTimezoneDropdownOpen(false);
										setTimezoneSearchTerm('');
									}}
								>
									<span className="trs-timezone-item-check">
										{timezone.value === tz.value && '✓'}
									</span>
									<span className="trs-timezone-item-name">
										({tz.offset}) {tz.name}
									</span>
								</button>
							))}
						</div>
					</div>
				)}
			</div>
		</div>
	);
}

export default TimeRangeSelector;
