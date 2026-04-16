import React, { useState, useCallback, useMemo, memo, useEffect, useRef } from 'react';
import {
  NetworkControlsProps,
  LayoutType,
  VisualMode
} from '../types';
import './NetworkControls.css';
import Typography from '../../../components/Typography/Typography';
import InputField from '../../../components/InputField/InputField';
import { arrowUp, arrowDown, checkBoxDarkmode, checkBoxLightmode, blankBoxDarkmode, blankBoxLightmode, searchSvg } from '../../../assets/ServiceMapIcons';
import { useTranslation } from 'react-i18next';
import { useIsDarkMode } from 'hooks/useDarkMode';

const NetworkControls: React.FC<NetworkControlsProps> = memo(({
  filters,
  onFiltersChange,
  isLoading,
  originalData
}) => {
  const isDarkMode = useIsDarkMode();
  const [isConnectionStatusOpen, setIsConnectionStatusOpen] = useState(true);
  const [isNamespaceOpen, setIsNamespaceOpen] = useState(true);
  const [protocolSearch, setProtocolSearch] = useState('');
  const [clusterSearch, setClusterSearch] = useState('');
  const [namespaceSearch, setNamespaceSearch] = useState('');
  const [workloadSearch, setWorkloadSearch] = useState('');
  const [tooltip, setTooltip] = useState<{ text: string; x: number; y: number } | null>(null);
  const tooltipTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const { t } = useTranslation('network_map');

  /** 필터 항목 hover 시 마우스 커서 아래에 fixed 툴팁 표시 */
  const handleFilterMouseEnter = useCallback((e: React.MouseEvent, text: string) => {
    const mouseX = e.clientX;
    const mouseY = e.clientY;
    tooltipTimerRef.current = setTimeout(() => {
      setTooltip({ text, x: mouseX + 12, y: mouseY + 16 });
    }, 150);
  }, []);

  const handleFilterMouseLeave = useCallback(() => {
    if (tooltipTimerRef.current) {
      clearTimeout(tooltipTimerRef.current);
      tooltipTimerRef.current = null;
    }
    setTooltip(null);
  }, []);
  const availableClusters = useMemo(() => {
    if (!originalData?.nodes || originalData.nodes.length === 0) return [];
    const clusters = [...new Set(originalData.nodes.map((node: any) => node.cluster))];
    return clusters.filter((cluster: string) => 
      cluster !== null && 
      cluster !== undefined && 
      cluster !== 'external-world'
    );
  }, [originalData]);

  const availableNamespaces = useMemo(() => {
    if (!originalData?.nodes || originalData.nodes.length === 0) return [];
    const namespaces = [...new Set(originalData.nodes.map((node: any) => node.namespace))];
    return namespaces.filter((namespace: string) => 
      namespace !== null && 
      namespace !== undefined && 
      namespace !== 'external-world'
    );
  }, [originalData]);

  const availableProtocols = useMemo(() => {
    if (!originalData?.edges || originalData.edges.length === 0) return [];
    const protocols = [...new Set(originalData.edges.map((edge: any) => edge.protocol))];
    return protocols.filter((protocol: string) => protocol !== null && protocol !== undefined);
  }, [originalData]);

  const availableWorkloads = useMemo(() => {
    if (!originalData?.nodes || originalData.nodes.length === 0) return [];
    const workloads = [...new Set(originalData.nodes.map((node: any) => node.workloadName))];
    return workloads.filter((workload: string) => workload && workload.length > 0);
  }, [originalData]);


  const availableConnectionStatuses = useMemo(() => {
    const statuses = ['Ok', 'Error']; // 백엔드 응답과 일치
    return statuses;
  }, []);

  const filteredProtocols = useMemo(() => {
    return availableProtocols.filter(protocol => 
      protocol.toLowerCase().includes(protocolSearch.toLowerCase())
    );
  }, [availableProtocols, protocolSearch]);

  const filteredClusters = useMemo(() => {
    return availableClusters.filter(cluster => 
      cluster.toLowerCase().includes(clusterSearch.toLowerCase())
    );
  }, [availableClusters, clusterSearch]);

  const filteredNamespaces = useMemo(() => {
    return availableNamespaces.filter(namespace => 
      namespace.toLowerCase().includes(namespaceSearch.toLowerCase())
    );
  }, [availableNamespaces, namespaceSearch]);

  const filteredWorkloads = useMemo(() => {
    return availableWorkloads.filter(workload => 
      workload.toLowerCase().includes(workloadSearch.toLowerCase())
    );
  }, [availableWorkloads, workloadSearch]);

  const toggleNamespace = useCallback((namespace: string) => {
    const newNamespaces = filters.namespaces.length === 1 && filters.namespaces.includes(namespace)
      ? [] // 이미 해당 항목만 선택된 상태면 모두 선택 (빈 배열)
      : [namespace]; // 그렇지 않으면 해당 항목만 선택
    
    onFiltersChange({
      ...filters,
      namespaces: newNamespaces,
    });
  }, [filters, onFiltersChange]);

  // ✅ 네임스페이스 체크박스 전용: 다중 선택 토글
  const toggleNamespaceCheckbox = useCallback((namespace: string, event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation(); // 상위 label 클릭 이벤트 차단
    
    let newNamespaces: string[];
    
    if (filters.namespaces.length === 0) {
      // 모든 항목이 선택된 상태에서 체크박스를 클릭하면, 클릭한 항목 제외
      newNamespaces = availableNamespaces.filter(n => n !== namespace);
    } else if (filters.namespaces.includes(namespace)) {
      // 이미 선택된 항목의 체크박스를 클릭하면 제거
      newNamespaces = filters.namespaces.filter(n => n !== namespace);
    } else {
      // 선택되지 않은 항목의 체크박스를 클릭하면 추가 (다중 선택)
      newNamespaces = [...filters.namespaces, namespace];
    }
    
    onFiltersChange({
      ...filters,
      namespaces: newNamespaces,
    });
  }, [filters, onFiltersChange, availableNamespaces]);

  const toggleProtocol = useCallback((protocol: string) => {
    const newProtocols = filters.protocols.length === 1 && filters.protocols.includes(protocol)
      ? []
      : [protocol];
    
    onFiltersChange({
      ...filters,
      protocols: newProtocols,
    });
  }, [filters, onFiltersChange]);

  const toggleProtocolCheckbox = useCallback((protocol: string, event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation(); 
    
    let newProtocols: string[];
    
    if (filters.protocols.length === 0) {
      newProtocols = availableProtocols.filter(p => p !== protocol);
    } else if (filters.protocols.includes(protocol)) {
      newProtocols = filters.protocols.filter(p => p !== protocol);
    } else {
      newProtocols = [...filters.protocols, protocol];
    }
    
    onFiltersChange({
      ...filters,
      protocols: newProtocols,
    });
  }, [filters, onFiltersChange, availableProtocols]);

  const toggleCluster = useCallback((cluster: string) => {
    const newClusters = filters.clusters.length === 1 && filters.clusters.includes(cluster)
      ? [] 
      : [cluster];
    
    onFiltersChange({
      ...filters,
      clusters: newClusters,
    });
  }, [filters, onFiltersChange]);

  const toggleClusterCheckbox = useCallback((cluster: string, event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    
    let newClusters: string[];
    
    if (filters.clusters.length === 0) {
      newClusters = availableClusters.filter(c => c !== cluster);
    } else if (filters.clusters.includes(cluster)) {
      newClusters = filters.clusters.filter(c => c !== cluster);
    } else {
      newClusters = [...filters.clusters, cluster];
    }
    
    onFiltersChange({
      ...filters,
      clusters: newClusters,
    });
  }, [filters, onFiltersChange, availableClusters]);

  const toggleShowErrors = useCallback(() => {
    onFiltersChange({ ...filters, showErrors: !filters.showErrors });
  }, [filters, onFiltersChange]);


  const toggleConnectionStatus = useCallback((status: string) => {
    const newConnectionStatuses = filters.connectionStatuses.length === 1 && filters.connectionStatuses.includes(status)
      ? [] 
      : [status];
    
    onFiltersChange({
      ...filters,
      connectionStatuses: newConnectionStatuses,
    });
  }, [filters, onFiltersChange]);

  const toggleConnectionStatusCheckbox = useCallback((status: string, event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    
    let newConnectionStatuses: string[];
    
    if (filters.connectionStatuses.length === 0) {
      newConnectionStatuses = availableConnectionStatuses.filter(s => s !== status);
    } else if (filters.connectionStatuses.includes(status)) {
      newConnectionStatuses = filters.connectionStatuses.filter(s => s !== status);
    } else {
      newConnectionStatuses = [...filters.connectionStatuses, status];
    }
    
    onFiltersChange({
      ...filters,
      connectionStatuses: newConnectionStatuses,
    });
  }, [filters, onFiltersChange, availableConnectionStatuses]);

  const toggleWorkload = useCallback((workload: string) => {
    const newWorkloads = filters.workloads.length === 1 && filters.workloads.includes(workload)
      ? [] 
      : [workload];
    
    onFiltersChange({
      ...filters,
      workloads: newWorkloads,
    });
  }, [filters, onFiltersChange]);

  const toggleWorkloadCheckbox = useCallback((workload: string, event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    
    let newWorkloads: string[];
    
    if (filters.workloads.length === 0) {
      newWorkloads = availableWorkloads.filter(w => w !== workload);
    } else if (filters.workloads.includes(workload)) {
      newWorkloads = filters.workloads.filter(w => w !== workload);
    } else {
      newWorkloads = [...filters.workloads, workload];
    }
    
    onFiltersChange({
      ...filters,
      workloads: newWorkloads,
    });
  }, [filters, onFiltersChange, availableWorkloads]);

  const handleOnlyAction = useCallback((filterType: string, value: string) => {
    switch (filterType) {
      case 'protocols':
        onFiltersChange({ ...filters, protocols: [value] });
        break;
      case 'clusters':
        onFiltersChange({ ...filters, clusters: [value] });
        break;
      case 'namespaces':
        onFiltersChange({ ...filters, namespaces: [value] });
        break;
      case 'workloads':
        onFiltersChange({ ...filters, workloads: [value] });
        break;
      case 'connectionStatuses':
        onFiltersChange({ ...filters, connectionStatuses: [value] });
        break;
    }
  }, [filters, onFiltersChange]);

  const handleAllAction = useCallback((filterType: string) => {
    switch (filterType) {
      case 'protocols':
        onFiltersChange({ ...filters, protocols: [] });
        break;
      case 'clusters':
        onFiltersChange({ ...filters, clusters: [] });
        break;
      case 'namespaces':
        onFiltersChange({ ...filters, namespaces: [] });
        break;
      case 'workloads':
        onFiltersChange({ ...filters, workloads: [] });
        break;
      case 'connectionStatuses':
        onFiltersChange({ ...filters, connectionStatuses: [] });
        break;
    }
  }, [filters, onFiltersChange]);



  return (
    <div className="network-controls">
      {isLoading && (
        <div className="controls-loading">
          <span className="loading-indicator">⟳ Loading...</span>
        </div>
      )}

      <div className="controls-main">
        <div className="filter-control-group">
          <div 
            className="accordion-header"
            onClick={() => setIsConnectionStatusOpen(!isConnectionStatusOpen)}
          >
            <Typography variant="b1" weight="regular" className="accordion-title">
              {t('common_filter')}
            </Typography>
            <img 
              className="accordion-arrow-icon"
              src={isConnectionStatusOpen ? arrowUp : arrowDown} 
              alt={isConnectionStatusOpen ? t('collapse') : t('expand')} 
            />
          </div>
          {isConnectionStatusOpen && (
            <>
              <div className="sub-filter-group">
                <div className="sub-filter-label" style={{paddingTop: '12px'}}>
                  <Typography variant="b2" weight="regular" className="sub-filter-text">
                    {t('connect_status')}
                  </Typography>
                </div>
                <div className="filter-options" role="group" aria-label={t('connection_status_filters')}>
            {availableConnectionStatuses.map((status: string) => {
              const isChecked = filters.connectionStatuses.length === 0 || filters.connectionStatuses.includes(status);
              const selectedCount = filters.connectionStatuses.length;
              
              let showOnlyButton = false;
              let showAllButton = false;
              
              if (selectedCount === 0) {
                if (isChecked) {
                  showOnlyButton = true;
                }
              } else if (selectedCount === 1) {
                if (isChecked) {
                  showAllButton = true;
                } else {
                  showOnlyButton = true; 
                }
              } else {
                if (isChecked) {
                  showOnlyButton = true;
                }
              }
              
              return (
                <div key={status} className="filter-option" onMouseEnter={(e) => handleFilterMouseEnter(e, status)} onMouseLeave={handleFilterMouseLeave}>
                  <div
                    className="checkbox-wrapper"
                    onClick={(e) => toggleConnectionStatusCheckbox(status, e)}
                  >
                    <img
                      src={isChecked ? (isDarkMode ? checkBoxDarkmode : checkBoxLightmode) : (isDarkMode ? blankBoxDarkmode : blankBoxLightmode)}
                      alt={isChecked ? "checked" : "unchecked"}
                      className="custom-checkbox"
                    />
                  </div>
                  <span
                    className="filter-label"
                    onClick={() => toggleConnectionStatus(status)}
                  >
                    {status}
                  </span>
                  {(showOnlyButton || showAllButton) && (
                    <button
                      className="filter-only-all-button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        if (showOnlyButton) {
                          handleOnlyAction('connectionStatuses', status);
                        } else if (showAllButton) {
                          handleAllAction('connectionStatuses');
                        }
                      }}
                    >
                      {showOnlyButton ? t('only') : t('all')}
                    </button>
                  )}
                </div>
              );
            })}
                </div>
              </div>

              <div className="sub-filter-group">
                <div className="sub-filter-label">
                  <Typography variant="b2" weight="regular" className="sub-filter-text">
                    {t('protocol')}
                  </Typography>
                </div>
                <div className="search-box">
                   <InputField
                        type="text"
                        placeholder={t('search', { ns: 'component_input' })}
                        leftDecoration={searchSvg}
                        inputText={protocolSearch}
                        onChange={(e) => {
                          setProtocolSearch(e.target.value);
                        }}
                        style={{
                          width: '100%',
                          height: '32px',
                          alignSelf: 'center',
                          backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff',
                        }}
                        placeholderStyle={{ backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff'}}
                        maxLength={253}
                    />
                </div>
                <div className="filter-options" role="group" aria-label={t('protocol_filters')}>
            {filteredProtocols.map((protocol: string) => {
              const isChecked = filters.protocols.length === 0 || filters.protocols.includes(protocol);
              const selectedCount = filters.protocols.length;
              
              let showOnlyButton = false;
              let showAllButton = false;
              
              if (selectedCount === 0) {
                if (isChecked) {
                  showOnlyButton = true;
                }
              } else if (selectedCount === 1) {
                if (isChecked) {
                  showAllButton = true; 
                } else {
                  showOnlyButton = true; 
                }
              } else {
                if (isChecked) {
                  showOnlyButton = true;
                }
              }
              
              
              return (
                <div key={protocol} className="filter-option" onMouseEnter={(e) => handleFilterMouseEnter(e, protocol)} onMouseLeave={handleFilterMouseLeave}>
                  <div
                    className="checkbox-wrapper"
                    onClick={(e) => toggleProtocolCheckbox(protocol, e)}
                  >
                    <img
                      src={isChecked ? (isDarkMode ? checkBoxDarkmode : checkBoxLightmode) : (isDarkMode ? blankBoxDarkmode : blankBoxLightmode)}
                      alt={isChecked ? "checked" : "unchecked"}
                      className="custom-checkbox"
                    />
                  </div>
                  <span
                    className="filter-label"
                    onClick={() => toggleProtocol(protocol)}
                  >
                    {protocol}
                  </span>
                  {(showOnlyButton || showAllButton) && (
                    <button
                      className="filter-only-all-button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        if (showOnlyButton) {
                          handleOnlyAction('protocols', protocol);
                        } else if (showAllButton) {
                          handleAllAction('protocols');
                        }
                      }}
                    >
                      {showOnlyButton ? t('only') : t('all')}
                    </button>
                  )}
                </div>
              );
            })}
                </div>
              </div>
            </>
          )}
        </div>
        <div className="filter-control-group">
          <div 
            className="accordion-header"
            onClick={() => setIsNamespaceOpen(!isNamespaceOpen)}
          >
            <Typography variant="b1" weight="regular" className="accordion-title">
              {t('k8s_filter')}
            </Typography>
            <img 
              src={isNamespaceOpen ? arrowUp : arrowDown} 
              alt={isNamespaceOpen ? t('collapse') : t('expand')} 
              className="accordion-arrow-icon"
            />
          </div>
          {isNamespaceOpen && (
            <>
              <div className="sub-filter-group">
                 <div className="sub-filter-label" style={{paddingTop: '12px'}}>
                  <Typography variant="b2" weight="regular" className="sub-filter-text">
                    {t('cluster')}
                  </Typography>
                </div>
                <div className="search-box">
                  <InputField
                        type="text"
                        placeholder={t('search', { ns: 'component_input' })}
                        leftDecoration={searchSvg}
                        inputText={clusterSearch}
                        onChange={(e) => {
                          setClusterSearch(e.target.value);
                        }}
                        style={{
                          width: '100%',
                          height: '32px',
                          alignSelf: 'center',
                          backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff',
                        }}
                        placeholderStyle={{ backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff'}}
                        maxLength={253}
                      />
                </div>
                <div className="filter-options" role="group" aria-label={t('cluster_filters')}>
                  {filteredClusters.map((cluster: string) => {
                    const isChecked = filters.clusters.length === 0 || filters.clusters.includes(cluster);
                    const selectedCount = filters.clusters.length;
                    
                    let showOnlyButton = false;
                    let showAllButton = false;
                    
                    if (selectedCount === 0) {
                      if (isChecked) {
                        showOnlyButton = true;
                      }
                    } else if (selectedCount === 1) {
                      if (isChecked) {
                        showAllButton = true; 
                      } else {
                        showOnlyButton = true;
                      }
                    } else {
                      if (isChecked) {
                        showOnlyButton = true;
                      }
                    }
                    
                    return (
                      <div key={cluster} className="filter-option" onMouseEnter={(e) => handleFilterMouseEnter(e, cluster)} onMouseLeave={handleFilterMouseLeave}>
                        <div
                          className="checkbox-wrapper"
                          onClick={(e) => toggleClusterCheckbox(cluster, e)}
                        >
                          <img
                            src={isChecked ? (isDarkMode ? checkBoxDarkmode : checkBoxLightmode) : (isDarkMode ? blankBoxDarkmode : blankBoxLightmode)}
                            alt={isChecked ? "checked" : "unchecked"}
                            className="custom-checkbox"
                          />
                        </div>
                        <span
                          className="filter-label"
                          onClick={() => toggleCluster(cluster)}
                        >
                          {cluster}
                        </span>
                        {(showOnlyButton || showAllButton) && (
                          <button
                            className="filter-only-all-button"
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              if (showOnlyButton) {
                                handleOnlyAction('clusters', cluster);
                              } else if (showAllButton) {
                                handleAllAction('clusters');
                              }
                            }}
                          >
                            {showOnlyButton ? t('only') : t('all')}
                          </button>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
              
              <div className="sub-filter-group">
                <div className="sub-filter-label">
                  <Typography variant="b2" weight="regular" className="sub-filter-text">
                    {t('namespace')}
                  </Typography>
                </div>
                <div className="search-box">
                  <InputField
                        type="text"
                        placeholder={t('search', { ns: 'component_input' })}
                        leftDecoration={searchSvg}
                        inputText={namespaceSearch}
                        onChange={(e) => {
                          setNamespaceSearch(e.target.value);
                        }}
                        style={{
                          width: '100%',
                          height: '32px',
                          alignSelf: 'center',
                          backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff',
                        }}
                        placeholderStyle={{ backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff'}}
                        maxLength={253}
                  />
                </div>
                <div className="filter-options" role="group" aria-label={t('namespace_filters')}>
                {filteredNamespaces.map((namespace: string) => {
                  const isChecked = filters.namespaces.length === 0 || filters.namespaces.includes(namespace);
                  const selectedCount = filters.namespaces.length;
              
              let showOnlyButton = false;
              let showAllButton = false;
              
              if (selectedCount === 0) {
                if (isChecked) {
                  showOnlyButton = true;
                }
              } else if (selectedCount === 1) {
                if (isChecked) {
                  showAllButton = true; 
                } else {
                  showOnlyButton = true;
                }
              } else {
                if (isChecked) {
                  showOnlyButton = true;
                }
              }
              
              return (
                <div key={namespace} className="filter-option" onMouseEnter={(e) => handleFilterMouseEnter(e, namespace)} onMouseLeave={handleFilterMouseLeave}>
                  <div
                    className="checkbox-wrapper"
                    onClick={(e) => toggleNamespaceCheckbox(namespace, e)}
                  >
                    <img
                      src={isChecked ? (isDarkMode ? checkBoxDarkmode : checkBoxLightmode) : (isDarkMode ? blankBoxDarkmode : blankBoxLightmode)}
                      alt={isChecked ? "checked" : "unchecked"}
                      className="custom-checkbox"
                    />
                  </div>
                  <span
                    className="filter-label"
                    onClick={() => toggleNamespace(namespace)}
                  >
                    {namespace}
                  </span>
                  {(showOnlyButton || showAllButton) && (
                    <button
                      className="filter-only-all-button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        if (showOnlyButton) {
                          handleOnlyAction('namespaces', namespace);
                        } else if (showAllButton) {
                          handleAllAction('namespaces');
                        }
                      }}
                    >
                      {showOnlyButton ? t('only') : t('all')}
                    </button>
                  )}
                </div>
              );
            })}
                </div>
              </div>
              
              <div className="sub-filter-group">
                <div className="sub-filter-label">
                  <Typography variant="b2" weight="regular" className="sub-filter-text">
                    {t('workload')}
                  </Typography>
                </div>
                <div className="search-box">
                 <InputField
                        type="text"
                        placeholder={t('search', { ns: 'component_input' })}
                        leftDecoration={searchSvg}
                        inputText={workloadSearch}
                        onChange={(e) => {
                          setWorkloadSearch(e.target.value);
                        }}
                        style={{
                          width: '100%',
                          height: '32px',
                          alignSelf: 'center',
                          backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff',
                        }}
                        placeholderStyle={{ backgroundColor: isDarkMode ? '#0b0c0e' : '#ffffff'}}
                        maxLength={253}
                    />
                </div>
                <div className="filter-options workload-filter" role="group" aria-label={t('workload_filters')}>
            {filteredWorkloads.map((workload: string) => {
              const isChecked = filters.workloads.length === 0 || filters.workloads.includes(workload);
              const selectedCount = filters.workloads.length;
              
              let showOnlyButton = false;
              let showAllButton = false;
              
              if (selectedCount === 0) {
                if (isChecked) {
                  showOnlyButton = true;
                }
              } else if (selectedCount === 1) {
                if (isChecked) {
                  showAllButton = true; 
                } else {
                  showOnlyButton = true; 
                }
              } else {
                if (isChecked) {
                  showOnlyButton = true;
                }
              }
              
              return (
                <div key={workload} className="filter-option" onMouseEnter={(e) => handleFilterMouseEnter(e, workload)} onMouseLeave={handleFilterMouseLeave}>
                  <div
                    className="checkbox-wrapper"
                    onClick={(e) => toggleWorkloadCheckbox(workload, e)}
                  >
                    <img
                      src={isChecked ? (isDarkMode ? checkBoxDarkmode : checkBoxLightmode) : (isDarkMode ? blankBoxDarkmode : blankBoxLightmode)}
                      alt={isChecked ? "checked" : "unchecked"}
                      className="custom-checkbox"
                    />
                  </div>
                  <span
                    className="filter-label"
                    onClick={() => toggleWorkload(workload)}
                  >
                    {workload}
                  </span>
                  {(showOnlyButton || showAllButton) && (
                    <button
                      className="filter-only-all-button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        if (showOnlyButton) {
                          handleOnlyAction('workloads', workload);
                        } else if (showAllButton) {
                          handleAllAction('workloads');
                        }
                      }}
                    >
                      {showOnlyButton ? t('only') : t('all')}
                    </button>
                  )}
                </div>
              );
            })}
                </div>
              </div>
            </>
          )}
        </div>
      </div>
      {tooltip && (
        <div
          className="filter-tooltip-fixed"
          style={{ left: tooltip.x, top: tooltip.y }}
        >
          {tooltip.text}
        </div>
      )}
    </div>
  );
});

NetworkControls.displayName = 'NetworkControls';

export default NetworkControls;