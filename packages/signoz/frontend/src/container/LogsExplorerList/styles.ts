import styled from 'styled-components';

export const InfinityWrapperStyled = styled.div<{ children?: React.ReactNode; 'data-testid'?: string }>`
	flex: 1;
	height: 40rem !important;
	display: flex;
	height: 100%;
`;
