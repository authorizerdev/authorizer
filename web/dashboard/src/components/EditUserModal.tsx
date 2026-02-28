import React, { useState } from 'react';
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
	Input,
} from '@chakra-ui/react';
import { useClient } from 'urql';
import { FaSave, FaPlus } from 'react-icons/fa';
import InputField from './InputField';
import {
	DateInputType,
	MultiSelectInputType,
	SelectInputType,
	TextInputType,
} from '../constants';
import { getObjectDiff, getGraphQLErrorMessage } from '../utils';
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
	roles: string[];
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
	const [newRole, setNewRole] = useState('');
	const { isOpen, onOpen, onClose } = useDisclosure();
	const [userData, setUserData] = useState<userDataTypes>({
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
	// Available roles for multiselect: current user roles (no env query)
	const availableRoles = Array.from(
		new Set([...(userData.roles || []), ...(user.roles || [])]),
	);
	React.useEffect(() => {
		setUserData(user);
	}, [user]);
	const saveHandler = async () => {
		const diff = getObjectDiff(user, userData);
		const updatedUserData = diff.reduce(
			(acc: any, property: string) => ({
				...acc,
				// @ts-ignore
				[property]: userData[property],
			}),
			{},
		);
		const res = await client
			.mutation(UpdateUser, { params: { ...updatedUserData, id: userData.id } })
			.toPromise();
		if (res.error) {
			toast({
				title: getGraphQLErrorMessage(res.error, 'User data update failed'),
				isClosable: true,
				status: 'error',
				position: 'top-right',
			});
		} else if (res.data?._update_user?.id) {
			toast({
				title: 'User data update successful',
				isClosable: true,
				status: 'success',
				position: 'top-right',
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
								<Center w="70%" flexDirection="column" alignItems="stretch">
									<InputField
										variables={userData}
										setVariables={setUserData}
										availableRoles={availableRoles}
										inputType={MultiSelectInputType.USER_ROLES}
									/>
									<Flex mt={2} gap={2}>
										<Input
											size="sm"
											placeholder="Add role"
											value={newRole}
											onChange={(e) => setNewRole(e.target.value)}
											onKeyDown={(e) => {
												if (e.key === 'Enter' && newRole.trim()) {
													setUserData({
														...userData,
														roles: [...(userData.roles || []), newRole.trim()],
													});
													setNewRole('');
												}
											}}
										/>
										<Button
											size="sm"
											leftIcon={<FaPlus />}
											onClick={() => {
												if (newRole.trim()) {
													setUserData({
														...userData,
														roles: [...(userData.roles || []), newRole.trim()],
													});
													setNewRole('');
												}
											}}
										>
											Add
										</Button>
									</Flex>
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
