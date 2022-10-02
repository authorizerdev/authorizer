import React from 'react';
import {
	Flex,
	Stack,
	Center,
	Text,
	useMediaQuery,
	Button,
	useToast,
} from '@chakra-ui/react';
import {
	HiddenInputType,
	TextInputType,
	TextAreaInputType,
} from '../../constants';
import GenerateKeysModal from '../GenerateKeysModal';
import InputField from '../InputField';
import { copyTextToClipboard } from '../../utils';

const JSTConfigurations = ({
	variables,
	setVariables,
	fieldVisibility,
	setFieldVisibility,
	SelectInputType,
	getData,
	HMACEncryptionType,
	RSAEncryptionType,
	ECDSAEncryptionType,
}: any) => {
	const [isNotSmallerScreen] = useMediaQuery('(min-width:600px)');
	const toast = useToast();

	const copyJSON = async () => {
		try {
			await copyTextToClipboard(
				JSON.stringify({
					type: variables.JWT_TYPE,
					key: variables.JWT_PUBLIC_KEY || variables.JWT_SECRET,
				}),
			);
			toast({
				title: `JWT config copied successfully`,
				isClosable: true,
				status: 'success',
				position: 'bottom-right',
			});
		} catch (err) {
			console.error({
				message: `Failed to copy JWT config`,
				error: err,
			});
			toast({
				title: `Failed to copy JWT config`,
				isClosable: true,
				status: 'error',
				position: 'bottom-right',
			});
		}
	};

	return (
		<div>
			{' '}
			<Flex
				borderRadius={5}
				width="100%"
				justifyContent="space-between"
				alignItems="center"
				paddingTop="2%"
			>
				<Text
					fontSize={isNotSmallerScreen ? 'md' : 'sm'}
					fontWeight="bold"
					mb={5}
				>
					JWT (JSON Web Tokens) Configurations
				</Text>
				<Flex mb={7}>
					<Button
						colorScheme="blue"
						h="1.75rem"
						size="sm"
						variant="ghost"
						onClick={copyJSON}
					>
						Copy As JSON Config
					</Button>
					<GenerateKeysModal jwtType={variables.JWT_TYPE} getData={getData} />
				</Flex>
			</Flex>
			<Stack spacing={6} padding="2% 0%">
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex w="30%" justifyContent="start" alignItems="center">
						<Text fontSize="sm">JWT Type:</Text>
					</Flex>
					<Flex
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '2'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={SelectInputType}
							value={SelectInputType}
							options={{
								...HMACEncryptionType,
								...RSAEncryptionType,
								...ECDSAEncryptionType,
							}}
						/>
					</Flex>
				</Flex>
				{Object.values(HMACEncryptionType).includes(variables.JWT_TYPE) ? (
					<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
						<Flex w="30%" justifyContent="start" alignItems="center">
							<Text fontSize="sm">JWT Secret</Text>
						</Flex>
						<Center
							w={isNotSmallerScreen ? '70%' : '100%'}
							mt={isNotSmallerScreen ? '0' : '2'}
						>
							<InputField
								borderRadius={5}
								variables={variables}
								setVariables={setVariables}
								fieldVisibility={fieldVisibility}
								setFieldVisibility={setFieldVisibility}
								inputType={HiddenInputType.JWT_SECRET}
							/>
						</Center>
					</Flex>
				) : (
					<>
						<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
							<Flex w="30%" justifyContent="start" alignItems="center">
								<Text fontSize="sm">Public Key</Text>
							</Flex>
							<Center
								w={isNotSmallerScreen ? '70%' : '100%'}
								mt={isNotSmallerScreen ? '0' : '2'}
							>
								<InputField
									borderRadius={5}
									variables={variables}
									setVariables={setVariables}
									inputType={TextAreaInputType.JWT_PUBLIC_KEY}
									placeholder="Add public key here"
									minH="25vh"
								/>
							</Center>
						</Flex>
						<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
							<Flex w="30%" justifyContent="start" alignItems="center">
								<Text fontSize="sm">Private Key</Text>
							</Flex>
							<Center
								w={isNotSmallerScreen ? '70%' : '100%'}
								mt={isNotSmallerScreen ? '0' : '2'}
							>
								<InputField
									borderRadius={5}
									variables={variables}
									setVariables={setVariables}
									inputType={TextAreaInputType.JWT_PRIVATE_KEY}
									placeholder="Add private key here"
									minH="25vh"
								/>
							</Center>
						</Flex>
					</>
				)}
				<Flex direction={isNotSmallerScreen ? 'row' : 'column'}>
					<Flex
						w={isNotSmallerScreen ? '30%' : '40%'}
						justifyContent="start"
						alignItems="center"
					>
						<Text fontSize="sm" orientation="vertical">
							JWT Role Claim:
						</Text>
					</Flex>
					<Center
						w={isNotSmallerScreen ? '70%' : '100%'}
						mt={isNotSmallerScreen ? '0' : '2'}
					>
						<InputField
							borderRadius={5}
							variables={variables}
							setVariables={setVariables}
							inputType={TextInputType.JWT_ROLE_CLAIM}
						/>
					</Center>
				</Flex>
			</Stack>
		</div>
	);
};

export default JSTConfigurations;
