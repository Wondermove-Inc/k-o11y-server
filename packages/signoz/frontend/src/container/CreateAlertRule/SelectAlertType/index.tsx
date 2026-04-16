import { Row, Tag, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import { ALERTS_DATA_SOURCE_MAP } from 'constants/alerts';
import { FeatureKeys } from 'constants/features';
import { useAppContext } from 'providers/App/App';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import { getOptionList } from './config';
import { AlertTypeCard, SelectTypeContainer } from './styles';
import { OptionType } from './types';

function SelectAlertType({ onSelect }: SelectAlertTypeProps): JSX.Element {
	const { t } = useTranslation(['alerts']);
	const { featureFlags } = useAppContext();

	const isAnomalyDetectionEnabled =
		featureFlags?.find((flag) => flag.name === FeatureKeys.ANOMALY_DETECTION)
			?.active || false;

	const optionList = getOptionList(t, isAnomalyDetectionEnabled);

	const renderOptions = useMemo(
		() => (
			<>
				{optionList.map((option: OptionType) => (
					<AlertTypeCard
						key={option.selection}
						title={option.title}
						extra={
							option.isBeta ? (
								<Tag bordered={false} color="geekblue">
									Beta
								</Tag>
							) : undefined
						}
						onClick={(): void => {
							onSelect(option.selection);
						}}
						data-testid={`alert-type-card-${option.selection}`}
					>
						{option.description}{' '}
					</AlertTypeCard>
				))}
			</>
		),
		[onSelect, optionList],
	);

	return (
		<SelectTypeContainer>
			<Typography.Title
				level={4}
				style={{
					padding: '0 8px',
				}}
			>
				{t('choose_alert_type')}
			</Typography.Title>
			<Row>{renderOptions}</Row>
		</SelectTypeContainer>
	);
}

interface SelectAlertTypeProps {
	onSelect: (typ: AlertTypes) => void;
}

export default SelectAlertType;
