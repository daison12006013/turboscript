interface Data {
    email: string;
    confirmationToken: string;
    name: string;
}

export const handle = async (event: Event): Promise<void> => {
    const data = event.body as unknown as Data;

    // The confirmation token is now in format: uid:timestamp.signature
    const verificationURL = `${event.env.APP_URL}/users/verify/${data.confirmationToken}`;

    // Use turboHtml to render the email template
    const htmlContent = await turboHtml('./app/routes/static/email-template.html', {
        Name: data.name,
        VerificationURL: verificationURL
    });

    await turboEmail({
        to: data.email,
        subject: 'Welcome to TurboScript - Verify Your Email',
        content: htmlContent,
        driver: 'smtp', // or 'mailgun', 'ses', 'sendgrid', 'postmark'
    });
};