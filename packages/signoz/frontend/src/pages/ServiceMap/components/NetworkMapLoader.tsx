import React from 'react';
import { useIsDarkMode } from 'hooks/useDarkMode';
import './NetworkMapLoader.css';

interface NetworkMapLoaderProps {
  message?: string;
  size?: 'small' | 'medium' | 'large';
}

const NetworkMapLoader: React.FC<NetworkMapLoaderProps> = ({
  message = 'Loading network topology...',
  size = 'medium'
}) => {
  const isDarkMode = useIsDarkMode();

  const sizeConfig = {
    small: 60,
    medium: 80,
    large: 120
  };

  const spinnerSize = sizeConfig[size];

  return (
    <div className={`network-map-loader ${isDarkMode ? 'dark' : 'light'}`}>
      <div className="network-map-loader-content">
        {/* CSS 스피너 */}
        <div
          className="network-map-loader-spinner"
          style={{
            width: `${spinnerSize}px`,
            height: `${spinnerSize}px`
          }}
        />

        {/* 로딩 메시지 */}
        <div className="network-map-loader-message">
          {message}
        </div>

        {/* 부가 정보 */}
        <div className="network-map-loader-sub-text">
          Analyzing nodes and connection data
        </div>
      </div>
    </div>
  );
};

export default NetworkMapLoader;