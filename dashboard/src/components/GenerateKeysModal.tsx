import React from 'react';
import {
	Button,
	Center,
	Flex,
	Modal,
	ModalBody,
	ModalCloseButton,
	ModalContent,
	ModalFooter,
	ModalHeader,
	ModalOverlay,
	useDisclosure,
	Text,
	useToast,
	Input,
} from '@chakra-ui/react';
import { useClient } from 'urql';
import { FaSave } from 'react-icons/fa';
import {
	ECDSAEncryptionType,
	envVarTypes,
	HMACEncryptionType,
	RSAEncryptionType,
	SelectInputType,
	TextAreaInputType,
} from '../constants';
import InputField from './InputField';

interface propTypes {
	saveEnvHandler: Function;
	variables: envVarTypes;
	setVariables: Function;
}

interface stateVarTypes {
	JWT_TYPE: string;
	JWT_SECRET: string;
	JWT_PRIVATE_KEY: string;
	JWT_PUBLIC_KEY: string;
}

const initState: stateVarTypes = {
	JWT_TYPE: '',
	JWT_SECRET: '',
	JWT_PRIVATE_KEY: '',
	JWT_PUBLIC_KEY: '',
};

const GenerateKeysModal = ({
	saveEnvHandler,
	variables,
	setVariables,
}: propTypes) => {
	const client = useClient();
	const toast = useToast();
	const { isOpen, onOpen, onClose } = useDisclosure();
	const [stateVariables, setStateVariables] = React.useState<stateVarTypes>({
		...initState,
	});
	React.useEffect(() => {
		if (isOpen) {
			setStateVariables({ ...initState, JWT_TYPE: variables.JWT_TYPE });
		}
	}, [isOpen]);
	const setKeys = () => {
		// fetch keys from api
		console.log('calling setKeys ==>> ', stateVariables.JWT_TYPE);
		if (true) {
			if (Object.values(HMACEncryptionType).includes(stateVariables.JWT_TYPE)) {
				setStateVariables({
					...stateVariables,
					JWT_SECRET: 'hello_world',
					JWT_PRIVATE_KEY: '',
					JWT_PUBLIC_KEY: '',
				});
			} else {
				setStateVariables({
					...stateVariables,
					JWT_SECRET: '',
					JWT_PRIVATE_KEY: 'test private key',
					JWT_PUBLIC_KEY: 'test public key',
				});
			}
			toast({
				title: 'New keys generated',
				isClosable: true,
				status: 'success',
				position: 'bottom-right',
			});
		} else {
			toast({
				title: 'Error occurred generating keys',
				isClosable: true,
				status: 'error',
				position: 'bottom-right',
			});
			closeHandler();
		}
	};
	React.useEffect(() => {
		if (isOpen) {
			setKeys();
		}
	}, [stateVariables.JWT_TYPE]);
	const saveHandler = async () => {
		setVariables({ ...variables, ...stateVariables });
		saveEnvHandler();
		closeHandler();
	};
	const closeHandler = async () => {
		setStateVariables({ ...initState });
		onClose();
	};
	return (
		<>
			<Button
				colorScheme="blue"
				h="1.75rem"
				size="sm"
				variant="ghost"
				onClick={onOpen}
			>
				Generate new keys
			</Button>
			<Modal isOpen={isOpen} onClose={onClose}>
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>New JWT keys</ModalHeader>
					<ModalCloseButton />
					<ModalBody>
						<Flex>
							<Flex w="30%" justifyContent="start" alignItems="center">
								<Text fontSize="sm">JWT Type:</Text>
							</Flex>
							<InputField
								variables={stateVariables}
								setVariables={setStateVariables}
								inputType={SelectInputType.JWT_TYPE}
								value={SelectInputType.JWT_TYPE}
								options={{
									...HMACEncryptionType,
									...RSAEncryptionType,
									...ECDSAEncryptionType,
								}}
							/>
						</Flex>
						{Object.values(HMACEncryptionType).includes(
							stateVariables.JWT_TYPE
						) ? (
							<Flex marginTop="8">
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">JWT Secret</Text>
								</Flex>
								<Center w="70%">
									<Input
										size="sm"
										value={stateVariables.JWT_SECRET}
										onChange={(event: any) =>
											setStateVariables({
												...stateVariables,
												JWT_SECRET: event.target.value,
											})
										}
									/>
								</Center>
							</Flex>
						) : (
							<>
								<Flex marginTop="8">
									<Flex w="30%" justifyContent="start" alignItems="center">
										<Text fontSize="sm">Public Key</Text>
									</Flex>
									<Center w="70%">
										<InputField
											variables={stateVariables}
											setVariables={setStateVariables}
											inputType={TextAreaInputType.JWT_PUBLIC_KEY}
											placeholder="Add public key here"
											minH="25vh"
										/>
									</Center>
								</Flex>
								<Flex marginTop="8">
									<Flex w="30%" justifyContent="start" alignItems="center">
										<Text fontSize="sm">Private Key</Text>
									</Flex>
									<Center w="70%">
										<InputField
											variables={stateVariables}
											setVariables={setStateVariables}
											inputType={TextAreaInputType.JWT_PRIVATE_KEY}
											placeholder="Add private key here"
											minH="25vh"
										/>
									</Center>
								</Flex>
							</>
						)}
					</ModalBody>

					<ModalFooter>
						<Button
							leftIcon={<FaSave />}
							colorScheme="red"
							variant="solid"
							onClick={saveHandler}
							isDisabled={false}
						>
							<Center h="100%" pt="5%">
								Apply
							</Center>
						</Button>
					</ModalFooter>
				</ModalContent>
			</Modal>
		</>
	);
};

export default GenerateKeysModal;
