import { CSSProperties, FunctionComponent, forwardRef } from 'react';
import './Typography.css';

interface Props {
  variant?:
    | 'h1'
    | 'h2'
    | 's1'
    | 'b1'
    | 'b2'
    | 'label'
    | 'link'
    | 'link-lowercase'
    | 'code';
  weight?: 'bold' | 'semi-bold' | 'medium' | 'regular' | 'thin';
  ellipsis?: boolean;
  style?: any;
  children?: any;
  link?: string;
  dangerouslySetInnerHTML?: any;
  onClick?: () => void;
  preserveNewlines?: boolean;
  className?: string;
}

const Typography = forwardRef<HTMLDivElement, Props>((props, ref) => {
  const {
    variant,
    weight,
    style,
    ellipsis,
    children,
    dangerouslySetInnerHTML,
    onClick,
    preserveNewlines = false,
    className,
  } = props;

  const defaultClassName = 'component-typography';
  const variantClassName = `${defaultClassName}-${
    variant === undefined ? 's1' : variant
  }`;
  const weightClassName = `${variantClassName}-${
    weight === undefined ? 'bold' : weight
  }`;
  const ellipsisClassName = `${defaultClassName}-${
    ellipsis === undefined ? 'non-ellipsis' : 'ellipsis'
  }`;

  const renderContent = (content: any) => {
    if (typeof content === 'string') {
      if (preserveNewlines) {
        return content.split('\n').map((line, index, array) => (
          <span key={index}>
            {line.trim()}
            {index < array.length - 1 && <br />}
          </span>
        ));
      }
      return content;
    }
    if (Array.isArray(content)) {
      return content.map((item, index) => (
        <span key={index}>
          {renderContent(item)}
          {index < content.length - 1 && <br />}
        </span>
      ));
    }
    return content;
  };

  return (
    <div
      className={`${defaultClassName} ${variantClassName} ${weightClassName} ${ellipsisClassName}${className ? ` ${className}` : ''}`}
      style={{ ...style }}
      onClick={onClick}
      ref={ref}
    >
      {renderContent(children)}
    </div>
  );
});

export default Typography;
