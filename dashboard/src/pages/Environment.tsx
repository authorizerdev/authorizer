import { Box, Divider, Flex } from '@chakra-ui/react';
import React from 'react';

// Don't allow changing database from here as it can cause persistence issues
export default function Environment() {
	return (
		<Box m="5" p="5" bg="white" rounded="md">
			<h1>Social Media Logins</h1>
			<Divider />- Add horizontal input for clientID and secret for - Google -
			Github - Facebook
			<h1>Roles</h1>
			<Divider />- Add tagged input for roles, default roles, and protected
			roles
			<h1>JWT Configurations</h1>
			<Divider />- Add input for JWT Type (keep this disabled for now with
			notice saying, "More JWT types will be enabled in upcoming releases"),JWT
			secret, JWT role claim
			<h1>Session Storage</h1>
			<Divider />- Add input for redis url
			<h1>Email Configurations</h1>
			<Divider />- Add input for SMTP Host, PORT, Username, Password, From
			Email,
			<h1>White Listing</h1>
			<Divider />- Add input for allowed origins
			<h1>Organization Information</h1>
			<Divider />- Add input for organization name, and logo
			<h1>Custom Scripts</h1>
			<Divider />- For now add text area input for CUSTOM_ACCESS_TOKEN_SCRIPT
			<h1>Disable Features</h1>
			<Divider />
			<h1>Danger</h1>
			<Divider />- Include changing admin secret
		</Box>
	);
}
