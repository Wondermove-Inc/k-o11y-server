/* eslint-disable sonarjs/cognitive-complexity */
/* eslint-disable jsx-a11y/click-events-have-key-events */
/* eslint-disable jsx-a11y/no-static-element-interactions */
/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable react/jsx-props-no-spreading */
/* eslint-disable no-nested-ternary */
import './styles.scss';

import { Input } from 'antd';
import { searchSvg } from 'assets/ServiceMapIcons';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import CustomSelect from './CustomSelect';
import { CustomSelectProps } from './types';

/**
 * CustomSelect with internal dropdown search functionality
 * This component wraps the original CustomSelect and adds a search input field
 * at the top of the dropdown for filtering options based on the search text.
 */
const CustomSelectWithSearch: React.FC<CustomSelectProps> = (propsWithDropdownRender) => {
	// Exclude dropdownRender and onSearch from props to avoid conflicts
	const { dropdownRender: _dropdownRender, onSearch: externalOnSearch, ...props } = propsWithDropdownRender;
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
	 * For single select, we don't filter the value
	 * Just pass it through as-is to maintain selection
	 */
	const filteredValue = props.value;

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
		if (externalOnSearch) {
			externalOnSearch(value);
		}
	}, [externalOnSearch]);

	// Override onSearch to sync with our internal search
	const handleSearch = useCallback((value: string): void => {
		// Ignore CustomSelect's internal search - we control it
		// Just forward to external handler if provided
		if (externalOnSearch) {
			externalOnSearch(value);
		}
	}, [externalOnSearch]);

	// Note: We don't override onDropdownVisibleChange to avoid breaking CustomSelect's internal state management
	// Clear search text when value changes (user made a selection)
	useEffect(() => {
		setInternalSearchText('');
		if (externalOnSearch) {
			externalOnSearch('');
		}
	}, [props.value, externalOnSearch]);

	// Custom dropdown render with search input at the top
	const customDropdownRender = useCallback((menu: React.ReactElement) => (
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
						// Prevent Enter from selecting - just for search filtering
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
	), [internalSearchText, handleInternalSearchChange]);

	return (
		<CustomSelect
			{...props}
			options={filteredOptions}
			value={filteredValue}
			noDataMessage={computedNoDataMessage}
			dropdownRender={customDropdownRender}
		/>
	);
};

export default CustomSelectWithSearch;
