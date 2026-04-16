import { Button as ButtonComponent } from 'antd';
import styled from 'styled-components';

export const RightActionContainer = styled.div`
	&&& {
		display: flex;
		align-items: center;
	}
`;

export const ButtonContainer = styled.div`
	&&& {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin: 0;
		padding-right: 0;
	}
`;

export const Button = styled(ButtonComponent)`
	&&& {
		margin-left: 1rem;
	}
`;
