/* eslint-disable sonarjs/cognitive-complexity */
/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-static-element-interactions */
/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable react/jsx-props-no-spreading */
/* eslint-disable no-nested-ternary */
import './styles.scss';

import { Input } from 'antd';
import { searchSvg } from 'assets/ServiceMapIcons';
import React, { useCallback, useMemo, useState } from 'react';

import CustomMultiSelect from './CustomMultiSelect';
import { CustomMultiSelectProps } from './types';

/**
 * CustomMultiSelect with internal dropdown search functionality
 * This component wraps the original CustomMultiSelect and adds a search input field
 * at the top of the dropdown for filtering options based on the search text.
 */
const CustomMultiSelectWithSearch: React.FC<CustomMultiSelectProps> = (propsWithDropdownRender) => {
	// Exclude dropdownRender from props to avoid conflicts
	const { dropdownRender: _dropdownRender, ...props } = propsWithDropdownRender;
	const [internalSearchText, setInternalSearchText] = useState('');

	// Filter options based on search text - only show matching options
	const filteredOptions = useMemo(() => {
		if (!internalSearchText || !props.options) {
			return props.options;
		}

		const searchLower = internalSearchText.toLowerCase();
		return props.options.filter((option) => {
			const label = option.label?.toString().toLowerCase() || '';
			const value = option.value?.toString().toLowerCase() || '';
			return label.includes(searchLower) || value.includes(searchLower);
		});
	}, [props.options, internalSearchText]);

	/**
	 * Filter selected values based on search text
	 * Only keep values that exist in filteredOptions
	 */
	const filteredValue = useMemo(() => {
		// No filtering needed if no search text or no value
		if (!internalSearchText || !props.value || !filteredOptions) {
			return props.value;
		}

		// Convert value to array for uniform processing
		const valueArray = Array.isArray(props.value) ? props.value : [props.value];

		// Create a set of filtered option values for fast lookup
		const filteredValueSet = new Set(filteredOptions.map((o) => o.value));

		// Keep only values that exist in filtered options
		const filtered = valueArray.filter((v) => filteredValueSet.has(v));

		// Return in the same format as the input (array or single value)
		return Array.isArray(props.value) ? filtered : filtered[0];
	}, [props.value, filteredOptions, internalSearchText]);

	/**
	 * Dynamically set noDataMessage based on search state
	 */
	const computedNoDataMessage = useMemo(() => {
		if (internalSearchText && filteredOptions && filteredOptions.length === 0) {
			return props.noDataMessage || 'No data found';
		}
		return props.noDataMessage;
	}, [internalSearchText, filteredOptions, props.noDataMessage]);

	// Handler for internal search input
	const handleInternalSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>): void => {
		const value = e.target.value;
		setInternalSearchText(value);

		// Call external onSearch handler if provided (for API search)
		if (props.onSearch) {
			props.onSearch(value);
		}
	}, [props]);

	// Clear internal search when dropdown closes
	const handleDropdownVisibleChange = useCallback((visible: boolean): void => {
		if (!visible) {
			setInternalSearchText('');

			// Reset external search when dropdown closes
			if (props.onSearch) {
				props.onSearch('');
			}
		}

		// Call original handler if provided
		if (props.onDropdownVisibleChange) {
			props.onDropdownVisibleChange(visible);
		}
	}, [props]);

	// Custom dropdown render with search input at the top
	const customDropdownRender = useCallback((menu: React.ReactElement) => {
		return (
			<div className="custom-multiselect-with-search-dropdown">
				{/* Search input field at the top */}
				<div className="dropdown-search-container">
					<img src={searchSvg} alt="search" className="dropdown-search-icon" />
					<Input
						placeholder="search"
						value={internalSearchText}
						onChange={handleInternalSearchChange}
						className="dropdown-search-input"
						bordered={false}
						autoFocus
						onMouseDown={(e): void => {
							e.stopPropagation();
						}}
						onClick={(e): void => {
							e.stopPropagation();
						}}
						onKeyDown={(e): void => {
							// Prevent Enter from adding values - just for search filtering
							if (e.key === 'Enter') {
								e.stopPropagation();
								e.preventDefault();
							}
						}}
					/>
				</div>

				{/* Original dropdown content with filtered options */}
				{menu}
			</div>
		);
	}, [internalSearchText, handleInternalSearchChange]);

	return (
		<CustomMultiSelect
			{...props}
			options={filteredOptions}
			value={filteredValue}
			noDataMessage={computedNoDataMessage}
			onDropdownVisibleChange={handleDropdownVisibleChange}
			dropdownRender={customDropdownRender}
			showSearch={false}
			enableRegexOption={false}
		/>
	);
};

export default CustomMultiSelectWithSearch;
