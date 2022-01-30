import React from 'react';
import {
	Button,
	Center,
	Flex,
	MenuItem,
	Modal,
	ModalBody,
	ModalCloseButton,
	ModalContent,
	ModalFooter,
	ModalHeader,
	ModalOverlay,
	Stack,
	useDisclosure,
	Text,
	useToast,
} from '@chakra-ui/react';
import { useClient } from 'urql';
import { FaSave } from 'react-icons/fa';
import InputField from './InputField';
import {
	ArrayInputType,
	DateInputType,
	SelectInputType,
	TextInputType,
} from '../constants';
import { getObjectDiff } from '../utils';
import { UpdateUser } from '../graphql/mutation';

const GenderTypes = {
	Undisclosed: null,
	Male: 'Male',
	Female: 'Female',
};

interface userDataTypes {
	id: string;
	email: string;
	given_name: string;
	family_name: string;
	middle_name: string;
	nickname: string;
	gender: string;
	birthdate: string;
	phone_number: string;
	picture: string;
	roles: [string] | [];
}

const EditUserModal = ({
	user,
	updateUserList,
}: {
	user: userDataTypes;
	updateUserList: Function;
}) => {
	const client = useClient();
	const toast = useToast();
	const { isOpen, onOpen, onClose } = useDisclosure();
	const [userData, setUserData] = React.useState<userDataTypes>({
		id: '',
		email: '',
		given_name: '',
		family_name: '',
		middle_name: '',
		nickname: '',
		gender: '',
		birthdate: '',
		phone_number: '',
		picture: '',
		roles: [],
	});
	React.useEffect(() => {
		setUserData(user);
	}, []);
	const saveHandler = async () => {
		const diff = getObjectDiff(user, userData);
		const updatedUserData = diff.reduce(
			(acc: any, property: string) => ({
				...acc,
				// @ts-ignore
				[property]: userData[property],
			}),
			{}
		);
		const res = await client
			.mutation(UpdateUser, { params: { ...updatedUserData, id: userData.id } })
			.toPromise();
		if (res.error) {
			toast({
				title: 'User data update failed',
				isClosable: true,
				status: 'error',
				position: 'bottom-right',
			});
		} else if (res.data?._update_user?.id) {
			toast({
				title: 'User data update successful',
				isClosable: true,
				status: 'success',
				position: 'bottom-right',
			});
		}
		onClose();
		updateUserList();
	};
	return (
		<>
			<MenuItem onClick={onOpen}>Edit User Details</MenuItem>
			<Modal isOpen={isOpen} onClose={onClose}>
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>Edit User Details</ModalHeader>
					<ModalCloseButton />
					<ModalBody>
						<Stack>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Given Name:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={TextInputType.GIVEN_NAME}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Middle Name:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={TextInputType.MIDDLE_NAME}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Family Name:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={TextInputType.FAMILY_NAME}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Birth Date:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={DateInputType.BIRTHDATE}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Nickname:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={TextInputType.NICKNAME}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Gender:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={SelectInputType.GENDER}
										value={userData.gender}
										options={GenderTypes}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Phone Number:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={TextInputType.PHONE_NUMBER}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Picture:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={TextInputType.PICTURE}
									/>
								</Center>
							</Flex>
							<Flex>
								<Flex w="30%" justifyContent="start" alignItems="center">
									<Text fontSize="sm">Roles:</Text>
								</Flex>
								<Center w="70%">
									<InputField
										variables={userData}
										setVariables={setUserData}
										inputType={ArrayInputType.USER_ROLES}
									/>
								</Center>
							</Flex>
						</Stack>
					</ModalBody>

					<ModalFooter>
						<Button
							leftIcon={<FaSave />}
							colorScheme="blue"
							variant="solid"
							onClick={saveHandler}
							isDisabled={false}
						>
							<Center h="100%" pt="5%">
								Save
							</Center>
						</Button>
					</ModalFooter>
				</ModalContent>
			</Modal>
		</>
	);
};

export default EditUserModal;
