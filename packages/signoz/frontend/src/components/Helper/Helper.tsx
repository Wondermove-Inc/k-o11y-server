import ReactDOM from 'react-dom';
import { useRef, useState } from 'react';
import './Helper.css';

interface Props {
  fixed?: boolean;
  children: any;
  title?: string;
  content?: string;
  information?: string;
  rightContent?: string;
  style?: any;
  titleStyle?: any;
}

const Helper = ({
  fixed = false,
  children,
  title,
  content,
  information,
  rightContent,
  style,
  titleStyle,
}: Props) => {
  const [isHovered, setIsHovered] = useState<boolean>(false);
  const childrenRef = useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = useState<{ left: number; top: number }>({
    left: 0,
    top: 0,
  });

  const handleHover = (e: React.MouseEvent<HTMLDivElement>) => {
    setIsHovered(true);
    setPosition({
      left: rightContent ? e.clientX - 530 : e.clientX,
      top: rightContent ? e.clientY - 404 : e.clientY,
    });
  };
  const handleMouseLeave = () => setIsHovered(false);

  const getSpeechBubbleStyle = () => {
    if (!childrenRef.current) return null;
    const contentLength = content?.length ? content.length * 10 : 0;
    const speechBubbleWidth = Math.min(contentLength, 240);
    return {
      width: `${speechBubbleWidth}px`,
    };
  };

  const getSpeechBubbleTop = () => {
    if (!childrenRef.current) return null;
    const childrenHeight = childrenRef.current.clientHeight;
    return {
      top: `${childrenHeight + 15}px`,
    };
  };


  const helperComponent = fixed ? (
    <>
      <div
        style={{
          ...style,
          display: 'inline-block',
          position: 'absolute',
        }}
      >
        <div ref={childrenRef}>{children}</div>
        {isHovered && rightContent && !content && !title && (
          <div className="helper-box-right" style={{ right: '30px' }}>
            <div className="helper-triangle-right" />
            <div className="helper-content-right" />
            {rightContent}
          </div>
        )}
        {isHovered && content && title && (
          <div className="helper-box-left">
            <div className="helper-triangle-left" />
            <div
              className="helper-content-left"
              style={{ ...getSpeechBubbleStyle() }}
            />
            <div style={titleStyle}>{title}</div>
            {Array.isArray(content) ? (
              content &&
              content?.map((item, index) => (
                <div
                  key={index}
                  style={{
                    display: 'flex',
                    marginLeft: '10px',
                    flexDirection: 'row',
                    gap: '8px',
                  }}
                >
                  <div style={{ display: 'flex', fontWeight: '900' }}>·</div>
                  <div style={{ display: 'flex' }}>{item}</div>
                </div>
              ))
            ) : (
              <div>{content}</div>
            )}
          </div>
        )}
        {isHovered && content && !title && (
          <div className="helper-fixed-container">
            <div className="helper-triangle-center" />
            <div
              className="helper-content-center"
              style={{ ...getSpeechBubbleStyle() }}
            />
            {content}
          </div>
        )}
        {isHovered && information && (
          <div className="helper-box-center">
            <div className="helper-triangle-center" />
            <div className="helper-content-center" />
            {information}
          </div>
        )}
      </div>
    </>
  ) : (
    <>
      <div
        style={{
          ...style,
          position: 'relative',
          display: 'inline-block',
        }}
        onMouseEnter={handleHover}
        onMouseLeave={handleMouseLeave}
      >
        <div ref={childrenRef}>{children}</div>
        {isHovered && rightContent && !content && !title && (
          <div
            className="helper-box-right"
            style={{
              right: '30px',
              // left: `${position.left}px`,
              // top: `${position.top + 20}px`,
            }}
          >
            <div className="helper-triangle-right" />
            <div className="helper-content-right" />
            {rightContent}
          </div>
        )}
        {isHovered && content && title && (
          <div
            className="helper-box-left"
            style={{
              left: `${position.left}px`,
              top: `${position.top + 20}px`,
            }}
          >
            <div className="helper-triangle-left" />
            <div
              className="helper-content-left"
              // style={{ ...getSpeechBubbleStyle() }}
            />
            <div style={titleStyle}>{title}</div>
            {Array.isArray(content) ? (
              content &&
              content?.map((item, index) => (
                <div
                  key={index}
                  style={{
                    display: 'flex',
                    marginLeft: '10px',
                    flexDirection: 'row',
                    gap: '8px',
                  }}
                >
                  <div style={{ display: 'flex', fontWeight: '900' }}>·</div>
                  <div style={{ display: 'flex' }}>{item}</div>
                </div>
              ))
            ) : (
              <div>{content}</div>
            )}
          </div>
        )}
        {isHovered && content && !title && (
          <div
            className="helper-box-center"
            style={{
              left: `${position.left}px`,
              top: `${position.top + 20}px`,
            }}
          >
            <div className="helper-triangle-center" />
            <div
              className="helper-content-center"
              // style={{ ...getSpeechBubbleStyle() }}
            />
            {content}
          </div>
        )}
        {isHovered && information && (
          <div
            className="helper-box-center"
            style={{
              left: `${position.left}px`,
              top: `${position.top + 20}px`,
            }}
          >
            <div className="helper-triangle-center" />
            <div className="helper-content-center" />
            {information}
          </div>
        )}
        {/* {isHovered && !rightContent && content && !title && (
        <div
          className="helper-box-left"
          style={{
            right: '0px',
            transform: 'translateX(7%)',
          }}
        >
          <div className="helper-triangle-center" />
          {content}
        </div>
      )} */}
      </div>
    </>
  );

  return (
    // <div
    //   style={{ position: 'relative', display: 'inline-block' }}
    //   onMouseEnter={handleHover}
    //   onMouseLeave={handleMouseLeave}
    // >
    //   <div ref={childrenRef}>{children}</div>
    //   {fixed
    //     ? ReactDOM.createPortal(helperComponent, document.body)
    //     : helperComponent}
    // </div>
    <>{helperComponent}</>
  );
};

export default Helper;
