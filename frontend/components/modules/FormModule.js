import React, { useState } from 'react';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ActivityIndicator, Alert } from 'react-native';

const API_URL = 'http://localhost:8080';

export default function FormModule({ module, eventId, userId, onComplete }) {
    const [answer, setAnswer] = useState('');
    const [submitting, setSubmitting] = useState(false);
    const [submitted, setSubmitted] = useState(false);

    const question = module.questions && module.questions.length > 0 ? module.questions[0] : null;

    const handleSubmit = async () => {
        if (!answer.trim()) {
            Alert.alert('Validation', 'Please enter an answer.');
            return;
        }

        setSubmitting(true);
        try {
            const response = await fetch(`${API_URL}/events/${eventId}/modules/${module.id}/response`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    userId: userId,
                    data: {
                        answer: answer,
                        questionId: question?.id
                    }
                }),
            });

            if (response.ok) {
                setSubmitted(true);
                if (onComplete) onComplete();
            } else {
                Alert.alert('Error', 'Failed to submit response.');
            }
        } catch (error) {
            console.error('Error submitting response:', error);
            Alert.alert('Error', 'Failed to submit response.');
        } finally {
            setSubmitting(false);
        }
    };

    if (submitted) {
        return (
            <View style={styles.container}>
                <Text style={styles.title}>Thank You!</Text>
                <Text style={styles.text}>Your answer has been submitted.</Text>
                <Text style={styles.subText}>Waiting for the host to continue...</Text>
            </View>
        );
    }

    if (!question) {
        return (
            <View style={styles.container}>
                <Text style={styles.errorText}>Invalid Form Configuration</Text>
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <Text style={styles.moduleTitle}>{module.name}</Text>
            <View style={styles.questionCard}>
                <Text style={styles.questionText}>{question.text}</Text>
                <TextInput
                    style={styles.input}
                    placeholder="Type your answer here..."
                    placeholderTextColor="#666"
                    value={answer}
                    onChangeText={setAnswer}
                    multiline
                />
                <TouchableOpacity
                    style={styles.submitButton}
                    onPress={handleSubmit}
                    disabled={submitting}
                >
                    {submitting ? (
                        <ActivityIndicator color="#FFF" />
                    ) : (
                        <Text style={styles.submitButtonText}>Submit Answer</Text>
                    )}
                </TouchableOpacity>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        width: '100%',
        alignItems: 'center',
    },
    moduleTitle: {
        fontSize: 24,
        fontWeight: 'bold',
        color: '#FFF',
        marginBottom: 20,
        textAlign: 'center',
    },
    questionCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 20,
        padding: 24,
        width: '100%',
        borderWidth: 1,
        borderColor: '#333',
    },
    questionText: {
        fontSize: 18,
        color: '#FFF',
        marginBottom: 20,
        lineHeight: 26,
    },
    input: {
        backgroundColor: '#2C2C2C',
        borderRadius: 12,
        padding: 16,
        fontSize: 16,
        color: '#FFF',
        marginBottom: 20,
        minHeight: 100,
        textAlignVertical: 'top',
    },
    submitButton: {
        backgroundColor: '#007AFF',
        paddingVertical: 16,
        borderRadius: 12,
        alignItems: 'center',
    },
    submitButtonText: {
        color: '#FFF',
        fontSize: 18,
        fontWeight: '600',
    },
    title: {
        fontSize: 28,
        fontWeight: 'bold',
        color: '#FFF',
        marginBottom: 10,
    },
    text: {
        fontSize: 18,
        color: '#CCC',
        marginBottom: 8,
    },
    subText: {
        fontSize: 14,
        color: '#888',
        fontStyle: 'italic',
    },
    errorText: {
        color: '#FF453A',
        fontSize: 16,
    },
});
