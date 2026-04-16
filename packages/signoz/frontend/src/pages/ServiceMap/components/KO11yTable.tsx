import React, { CSSProperties } from 'react';
import { Table as AntdTable } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import './KO11yTable.css';

export interface TableHeader {
  name: string;
  accessor: string;
  minWidth?: number;
  maxWidth?: number;
  textAlign?: 'left' | 'center' | 'right';
  showTooltip?: boolean;
}

export interface TableItems<T = any> {
  headers: TableHeader[];
  body: T[];
}

export interface KO11yTableProps<T = any> {
  items: TableItems<T>;
  tappable?: boolean;
  footer?: boolean;
  firstCellDivider?: boolean;
  tableHeight?: string | number;
  bodyStyle?: CSSProperties;
}

function KO11yTable<T extends Record<string, any>>({
  items,
  tappable = false,
  footer = false,
  firstCellDivider = false,
  tableHeight = 'auto',
  bodyStyle = {}
}: KO11yTableProps<T>): JSX.Element {
  // Convert headers to antd columns format
  const columns: ColumnsType<T> = items.headers.map((header, index) => ({
    title: header.name,
    dataIndex: header.accessor,
    key: header.accessor,
    align: header.textAlign || 'left',
    width: header.minWidth,
    ellipsis: header.showTooltip ? { showTitle: true } : false,
    render: (value: any, record: T) => {
      // Handle special rendering for httpStatusCode
      if (header.accessor === 'httpStatusCode' && typeof value === 'object') {
        const { status, protocol, isError } = value;

        // ✅ 최우선: 백엔드 is_error 플래그 체크 (OpenTelemetry span status 포함)
        // status >= 400은 isError가 undefined인 경우의 fallback
        const statusClass =
          isError !== undefined
            ? (isError ? 'status-error' : 'status-success')  // 백엔드 판단 사용
            : (status >= 400 ? 'status-error' : status >= 300 ? 'status-warning' : 'status-success');  // Fallback

        return <span className={`status-badge ${statusClass}`}>{status}</span>;
      }

      // Handle React elements (like PathCell component)
      if (React.isValidElement(value)) {
        return value;
      }

      return value;
    }
  }));

  // Add keys to data items
  const dataWithKeys = items.body.map((item, index) => ({
    ...item,
    key: `row-${index}`
  }));

  const scrollConfig = tableHeight !== 'auto' ? { y: typeof tableHeight === 'string' ? parseInt(tableHeight) : tableHeight } : undefined;

  return (
    <div className="ko11y-table-wrapper" style={bodyStyle}>
      <AntdTable<T>
        columns={columns}
        dataSource={dataWithKeys}
        pagination={false}
        scroll={scrollConfig}
        showHeader={true}
        bordered={false}
        size="small"
        className={`ko11y-table ${firstCellDivider ? 'first-cell-divider' : ''} ${tappable ? 'tappable' : ''}`}
      />
      {footer && <div className="ko11y-table-footer"></div>}
    </div>
  );
}

export default KO11yTable;
