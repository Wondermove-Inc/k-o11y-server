import { K8sRolloutsData } from 'api/infraMonitoring/getK8sRolloutsList';

export type RolloutDetailsProps = {
	rollout: K8sRolloutsData | null;
	isModalTimeSelection: boolean;
	onClose: () => void;
};
