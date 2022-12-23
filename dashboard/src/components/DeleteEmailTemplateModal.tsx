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
import { DeleteEmailTemplate } from '../graphql/mutation';
import { capitalizeFirstLetter } from '../utils';

interface deleteEmailTemplateModalInputPropTypes {
	emailTemplateId: string;
	eventName: string;
	fetchEmailTemplatesData: Function;
}

const DeleteEmailTemplateModal = ({
	emailTemplateId,
	eventName,
	fetchEmailTemplatesData,
}: deleteEmailTemplateModalInputPropTypes) => {
	const client = useClient();
	const toast = useToast();
	const { isOpen, onOpen, onClose } = useDisclosure();

	const deleteHandler = async () => {
		const res = await client
			.mutation(DeleteEmailTemplate, { params: { id: emailTemplateId } })
			.toPromise();
		if (res.error) {
			toast({
				title: capitalizeFirstLetter(res.error.message),
				isClosable: true,
				status: 'error',
				position: 'top-right',
			});

			return;
		} else if (res.data?._delete_email_template) {
			toast({
				title: capitalizeFirstLetter(res.data?._delete_email_template.message),
				isClosable: true,
				status: 'success',
				position: 'top-right',
			});
		}
		onClose();
		fetchEmailTemplatesData();
	};
	return (
		<>
			<MenuItem onClick={onOpen}>Delete</MenuItem>
			<Modal isOpen={isOpen} onClose={onClose}>
				<ModalOverlay />
				<ModalContent>
					<ModalHeader>Delete Email Template</ModalHeader>
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
								Email template for event <b>{eventName}</b> will be deleted
								permanently!
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

export default DeleteEmailTemplateModal;
