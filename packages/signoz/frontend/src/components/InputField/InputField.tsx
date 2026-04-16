import {
  ChangeEvent,
  FunctionComponent,
  useState,
  FormEvent,
  useEffect,
} from 'react';

import Helper from '../Helper/Helper';
import Typography from '../Typography/Typography';

import './InputField.css';
import { useTranslation } from 'react-i18next';
import { triangleDownSvg, triangleUpSvg } from '../../assets/ServiceMapIcons';
import { useIsDarkMode } from 'hooks/useDarkMode';

/**
 * onPressEnter : 엔터 눌렀을 때의 동작 정의
 */
interface Props {
  style?: any;
  maxLength?: number;
  inputStyle?: any;
  inputText?: string;
  children?: any;
  disabled?: boolean;
  leftDecoration?: string;
  leftTextDecoration?: string;
  rightDecoration?: string;
  secondRightDecoration?: string;
  rightStyle?: any;
  secondRightStyle?: any;
  placeholderStyle?: any;
  valid?: boolean;
  rightDecorationTapped?: (
    e: React.MouseEvent<HTMLImageElement, MouseEvent>,
  ) => void;
  secondRightDecorationTapped?: (
    e: React.MouseEvent<HTMLImageElement, MouseEvent>,
  ) => void;

  // rightDecorationTapped?: () => void;
  // secondRightDecorationTapped?: () => void;

  type?:
    | 'email'
    | 'text'
    | 'password'
    | 'number'
    | 'uid'
    | 'box'
    | 'custom-time';
  helperText?: string;
  secondHelperText?: string;

  error?: boolean;
  errorText?: string;

  verifyText?: string;
  placeholder?: string | number | null;
  defaultValue?: any;
  autoFocus?: boolean;

  min?: number;
  max?: number;

  onChange?: (e: ChangeEvent<HTMLInputElement>) => void;
  onPressEnter?: (value: string) => void;
  onInvalid?: (e: FormEvent<HTMLInputElement>) => void;
  //
  // setInputValue?: any;
  // nodeSize?: string;
}

const InputField: FunctionComponent<Props> = (props: Props) => {
  const {
    style,
    maxLength,
    inputStyle,
    inputText,
    // setInputValue,
    // nodeSize,
    disabled = false,
    defaultValue,
    leftDecoration,
    leftTextDecoration,
    type,
    children,
    rightDecoration,
    placeholderStyle,
    secondRightDecoration,
    secondRightStyle,
    rightStyle,
    error,
    errorText,
    helperText,
    secondHelperText,
    verifyText,
    placeholder,
    autoFocus,
    min,
    max,
    valid,
    onChange,
    onPressEnter,
    onInvalid,
    rightDecorationTapped,
    secondRightDecorationTapped,
  } = props;
  const isDarkMode = useIsDarkMode();
  const errorTextStyle: React.CSSProperties = {
    color: 'var(--status-danger)',
    marginTop: '-18px',
    fontSize: 'var(--body-label-bold-size)',
    fontWeight: '400',
    position: 'absolute',
    top: '57px',
    left: '-1px',
  };

  const formatTimeInput = (value: string) => {
    if (/^\d{4}$/.test(value)) {
      const hours = Math.min(23, parseInt(value.slice(0, 2), 10)); // 시간은 23을 초과하지 않음
      const minutes = Math.min(59, parseInt(value.slice(2), 10)); // 분은 59를 초과하지 않음
      return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(
        2,
        '0',
      )}`; // 00:00 형식으로 반환
    }
    return value;
  };

  /**
   * 스토리북에서 엔터키 눌렀을때 폼 전송되는 것 방지
   * @param event
   */
  const handleSubmit = (event: any) => event.preventDefault();
  const [focused, setFocused] = useState<string>(
    (inputText && inputText.length > 0) || (defaultValue && defaultValue.length)
      ? 'focused'
      : 'unfocused',
  );
  const [inputValue, setInputValue] = useState<string>(
    type === 'number' ? inputText || '1' : defaultValue,
  );
  const defaultClassName = 'component-inputfield';
  const labelClassName = `${defaultClassName}-label${
    leftDecoration ? '-left-deco' : ''
  }`;

  useEffect(() => {
    setInputValue(inputText || '');
  }, [inputText]);

  useEffect(() => {
    if (disabled) setFocused('focused');
  }, [disabled]);

  const handleChange = (e: any) => {
    if (type === 'custom-time') {
      let newValue = e.target.value;
      newValue = newValue.replace(/[^0-9]/g, ''); // 숫자 이외의 입력 제거
      if (newValue.length > 4) newValue = newValue.slice(0, 4); // 최대 4자리까지만 입력 허용
      newValue = formatTimeInput(newValue); // 시간 형식으로 변환

      if (onChange) onChange(e);
      setInputValue(newValue);
    } else {
      const newValue =
        type === 'number'
          ? Number(e.target.value) >= 0
            ? e.target.value
            : '0'
          : e.target.value;
      if (onChange) onChange(e);

      if (max && newValue > max) {
        setInputValue(String(max));
      } else {
        setInputValue(newValue);
      }
    }
  };

  const handleFocus = () => {
    setFocused('focused');
  };

  const handleBlur = (e: any) => {
    if (type === 'custom-time' && !e.target.value) {
      setInputValue('00:00');
      return;
    }
    if (e.target.value.length > 0) {
      setFocused('focused');
    } else {
      setFocused('unfocused');
    }
  };

  const incrementValue = () => {
    const newValue = String(Number(inputValue) + 1);
    setInputValue(newValue);

    if (max && Number(inputValue) >= max) {
      setInputValue(String(max));
    }

    if (onChange)
      onChange({
        target: { value: newValue },
      } as ChangeEvent<HTMLInputElement>);
  };

  const decrementValue = () => {
    const currentValue = Number(inputValue);
    const minValue = min || 0;
    const newValue = Math.max(minValue, currentValue - 1);
    const newValueString = String(newValue);
    
    setInputValue(newValueString);

    if (onChange)
      onChange({
        target: { value: newValueString },
      } as ChangeEvent<HTMLInputElement>);
  };

  return (
    <>
      <form
        className={`${defaultClassName}`}
        style={
          !valid
            ? style
            : {
                ...style,
                border: '1px solid var(--status-danger)',
                marginBottom: '5px',
              }
        }
        onSubmit={handleSubmit}
      >
        {leftDecoration && <img src={leftDecoration} alt="" />}
        {leftTextDecoration && (
          <Typography variant="b1" weight="thin">
            {leftTextDecoration}
          </Typography>
        )}
        <div className={`${labelClassName}-${focused}`} onClick={handleFocus}>
          <Typography
            variant="label"
            weight="regular"
            style={{
              color: isDarkMode ? 'var(--text-tertiary)' : 'rgba(0, 0, 0, 0.5)',
              // backgroundColor: 'rgb(38,41,53)',
              // marginTop: '1px',
              ...placeholderStyle,
            }}
          >
            {placeholder}
          </Typography>
        </div>
        <input
          className={`${defaultClassName}-input`}
          defaultValue={defaultValue}
          disabled={disabled}
          value={inputValue}
          autoFocus={autoFocus || focused === 'focused'}
          type={!type ? 'text' : type === 'uid' ? 'text' : type}
          min={type === 'number' && min ? min : undefined}
          max={type === 'number' && max ? max : undefined}
          onChange={(e: ChangeEvent<HTMLInputElement>) =>
            handleChange && handleChange(e)
          }
          onFocus={handleFocus}
          maxLength={maxLength}
          onBlur={(e: React.FocusEvent<HTMLInputElement>) =>
            handleBlur && handleBlur(e)
          }
          style={{
            ...inputStyle,
          }}
          onInvalid={onInvalid ? onInvalid : () => {}}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && onPressEnter) onPressEnter(inputValue);
            // onPressEnter(String(Number(inputText)));
          }}
        />
        {type === 'number' && !disabled && (
          <div className="number-spinner">
            <div
              // type="button"
              className="increment"
              // onClick={() => setInputValue(String(Number(inputValue) + 1))}
              onClick={incrementValue}
            >
              <img src={triangleUpSvg} alt="up" style={{ width: '10px' }} />
            </div>
            <div
              className="decrement"
              // onClick={() => setInputValue(String(Number(inputValue) - 1))}
              onClick={decrementValue}
            >
              <img src={triangleDownSvg} alt="down" style={{ width: '10px' }} />
            </div>
          </div>
        )}
        {rightDecoration && helperText ? (
          <Helper rightContent={helperText} style={{}}>
            <img
              src={rightDecoration}
              style={rightStyle}
              alt=""
              className="input-right-deco"
              onClick={rightDecorationTapped}
            />
          </Helper>
        ) : (
          <img
            src={rightDecoration}
            style={rightStyle}
            alt=""
            className="input-right-deco"
            onClick={rightDecorationTapped}
          />
        )}
        {secondRightDecoration && secondHelperText ? (
          <Helper rightContent={secondHelperText} style={{}}>
            <img
              src={secondRightDecoration}
              alt=""
              style={secondRightStyle}
              onClick={secondRightDecorationTapped}
            />
          </Helper>
        ) : (
          <img
            src={secondRightDecoration}
            alt=""
            style={secondRightStyle}
            onClick={secondRightDecorationTapped}
          />
        )}

        {children}
        {error && <Typography style={errorTextStyle}>{errorText}</Typography>}
      </form>
    </>
  );
};

export default InputField;
