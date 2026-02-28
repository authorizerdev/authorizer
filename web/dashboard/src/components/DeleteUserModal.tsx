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
	useDisclosure,
	Text,
	useToast,
} from '@chakra-ui/react';
import { useClient } from 'urql';
import { FaRegTrashAlt } from 'react-icons/fa';
import { DeleteUser } from '../graphql/mutation';
import { capitalizeFirstLetter, getGraphQLErrorMessage } from '../utils';

interface userDataTypes {
	id: string;
	email: string;
}

const DeleteUserModal = ({
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
	});
	React.useEffect(() => {
		setUserData(user);
	}, []);
	const deleteHandler = async () => {
		const res = await client
			.mutation(DeleteUser, { params: { email: userData.email } })
			.toPromise();
		if (res.error) {
			toast({
				title: capitalizeFirstLetter(getGraphQLErrorMessage(res.error, 'Failed to delete user')),
				isClosable: true,
				status: 'error',
				position: 'top-right',
			});

			return;
		} else if (res.data?._delete_user) {
			toast({
				title: capitalizeFirstLetter(res.data?._delete_user.message),
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
			<MenuItem onClick={onOpen}>Delete User</MenuItem>
			<Modal isOpen={isOpen} onClose={onClose}>
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>Delete User</ModalHeader>
					<ModalCloseButton />
					<ModalBody>
						<Text fontSize="md">Are you sure?</Text>
						<Flex
							padding="5%"
							marginTop="5%"
							marginBottom="2%"
							border="1px solid #ff7875"
							borderRadius="5px"
							flexDirection="column"
						>
							<Text fontSize="sm">
								User <b>{user.email}</b> will be deleted permanently!
							</Text>
						</Flex>
					</ModalBody>

					<ModalFooter>
						<Button
							leftIcon={<FaRegTrashAlt />}
							colorScheme="red"
							variant="solid"
							onClick={deleteHandler}
							isDisabled={false}
						>
							<Center h="100%" pt="5%">
								Delete
							</Center>
						</Button>
					</ModalFooter>
				</ModalContent>
			</Modal>
		</>
	);
};

export default DeleteUserModal;
