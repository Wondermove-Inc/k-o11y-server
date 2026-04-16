import styled from 'styled-components';

export const TitleWrapper = styled.span<{ children?: React.ReactNode; onMouseDown?: (e: React.MouseEvent) => void }>`
	user-select: text !important;
	cursor: text;

	.hover-reveal {
		visibility: hidden;
	}

	&:hover .hover-reveal {
		visibility: visible;
	}
`;
