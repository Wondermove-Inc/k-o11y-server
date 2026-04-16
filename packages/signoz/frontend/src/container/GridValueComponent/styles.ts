import styled from 'styled-components';

interface Props {
	isDashboardPage: boolean;
	children?: React.ReactNode;
}

interface ValueContainerProps {
	showClickable?: boolean;
	children?: React.ReactNode;
	onClick?: (e: React.MouseEvent) => void;
}

export const ValueContainer = styled.div<ValueContainerProps>`
	height: 100%;
	display: flex;
	justify-content: center;
	align-items: center;
	flex-direction: column;
	user-select: none;
	cursor: ${({ showClickable = false }): string =>
		showClickable ? 'pointer' : 'default'};
`;

export const TitleContainer = styled.div<Props>`
	text-align: center;
	padding-top: 0;
	margin-top: -8px;
`;
