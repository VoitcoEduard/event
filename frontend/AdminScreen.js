import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, TextInput, Alert, Platform } from 'react-native';
import { StatusBar } from 'expo-status-bar';

const API_URL = 'http://localhost:8080';

const EventCard = ({ event, onEdit, onDelete }) => (
    <View style={styles.eventCard}>
        <View style={styles.cardHeader}>
            <Text style={styles.eventName}>{event.name}</Text>
            <Text style={styles.eventId}>ID: {event.id}</Text>
        </View>
        <Text style={styles.eventDescription}>{event.description}</Text>
        <View style={styles.cardActions}>
            <TouchableOpacity style={[styles.cardAction, styles.editAction]} onPress={() => onEdit(event)}>
                <Text style={styles.cardActionText}>Edit</Text>
            </TouchableOpacity>
            <TouchableOpacity style={[styles.cardAction, styles.deleteAction]} onPress={() => onDelete(event.id)}>
                <Text style={[styles.cardActionText, styles.deleteText]}>Delete</Text>
            </TouchableOpacity>
        </View>
    </View>
);

export default function AdminScreen({ onNavigate }) {
    const [events, setEvents] = useState([]);
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const [editingId, setEditingId] = useState(null);

    useEffect(() => {
        fetchEvents();
    }, []);

    const fetchEvents = async () => {
        try {
            const response = await fetch(`${API_URL}/events`);
            const data = await response.json();
            setEvents(data || []);
        } catch (error) {
            console.error('Error fetching events:', error);
            Alert.alert('Error', 'Failed to fetch events');
        }
    };

    const handleSubmit = async () => {
        if (!name || !description) {
            Alert.alert('Error', 'Please fill in all fields');
            return;
        }

        try {
            const url = editingId ? `${API_URL}/events/${editingId}` : `${API_URL}/events`;
            const method = editingId ? 'PUT' : 'POST';

            const response = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, description }),
            });

            if (response.ok) {
                setName('');
                setDescription('');
                setEditingId(null);
                fetchEvents();
                Alert.alert('Success', `Event ${editingId ? 'updated' : 'created'} successfully`);
            } else {
                Alert.alert('Error', 'Failed to save event');
            }
        } catch (error) {
            console.error('Error saving event:', error);
            Alert.alert('Error', 'Failed to save event');
        }
    };

    const handleDelete = async (id) => {
        if (Platform.OS === 'web') {
            if (confirm('Are you sure you want to delete this event?')) {
                await performDelete(id);
            }
        } else {
            Alert.alert(
                'Delete Event',
                'Are you sure you want to delete this event?',
                [
                    { text: 'Cancel', style: 'cancel' },
                    { text: 'Delete', style: 'destructive', onPress: () => performDelete(id) },
                ]
            );
        }
    };

    const performDelete = async (id) => {
        try {
            const response = await fetch(`${API_URL}/events/${id}`, { method: 'DELETE' });
            if (response.ok) {
                fetchEvents();
            } else {
                Alert.alert('Error', 'Failed to delete event');
            }
        } catch (error) {
            console.error('Error deleting event:', error);
            Alert.alert('Error', 'Failed to delete event');
        }
    };

    const handleEdit = (event) => {
        setName(event.name);
        setDescription(event.description);
        setEditingId(event.id);
    };

    return (
        <View style={styles.container}>
            <StatusBar style="light" />

            <View style={styles.header}>
                <Text style={styles.headerTitle}>Event Admin</Text>
                <Text style={styles.headerSubtitle}>Manage Events Listings</Text>
            </View>

            <ScrollView
                style={styles.content}
                contentContainerStyle={styles.scrollContent}
                showsVerticalScrollIndicator={false}
            >
                <View style={styles.formCard}>
                    <Text style={styles.formTitle}>{editingId ? 'Edit Event' : 'New Event'}</Text>
                    <TextInput
                        style={styles.input}
                        placeholder="Event Name"
                        placeholderTextColor="#666"
                        value={name}
                        onChangeText={setName}
                    />
                    <TextInput
                        style={[styles.input, styles.textArea]}
                        placeholder="Description"
                        placeholderTextColor="#666"
                        value={description}
                        onChangeText={setDescription}
                        multiline
                        numberOfLines={3}
                    />
                    <View style={styles.formActions}>
                        {editingId && (
                            <TouchableOpacity
                                style={styles.cancelButton}
                                onPress={() => {
                                    setEditingId(null);
                                    setName('');
                                    setDescription('');
                                }}
                            >
                                <Text style={styles.cancelButtonText}>Cancel</Text>
                            </TouchableOpacity>
                        )}
                        <TouchableOpacity style={styles.submitButton} onPress={handleSubmit}>
                            <Text style={styles.submitButtonText}>{editingId ? 'Update Event' : 'Create Event'}</Text>
                        </TouchableOpacity>
                    </View>
                </View>

                <Text style={styles.sectionTitle}>Upcoming Events</Text>

                <View style={styles.eventsGrid}>
                    {events.map((event) => (
                        <EventCard
                            key={event.id}
                            event={event}
                            onEdit={handleEdit}
                            onDelete={handleDelete}
                        />
                    ))}
                    {events.length === 0 && (
                        <Text style={styles.emptyText}>No events found. Create one above!</Text>
                    )}
                </View>

                <TouchableOpacity
                    style={styles.backButton}
                    onPress={() => onNavigate('main')}
                    activeOpacity={0.8}
                >
                    <Text style={styles.backButtonText}>Back to Main</Text>
                </TouchableOpacity>
            </ScrollView>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
    },
    header: {
        paddingTop: 60,
        paddingBottom: 20,
        paddingHorizontal: 24,
        backgroundColor: '#1E1E1E',
        borderBottomWidth: 1,
        borderBottomColor: '#2C2C2C',
    },
    headerTitle: {
        fontSize: 28,
        fontWeight: '800',
        color: '#FFFFFF',
        letterSpacing: -0.5,
        marginBottom: 4,
    },
    headerSubtitle: {
        fontSize: 14,
        color: '#888888',
        textTransform: 'uppercase',
        letterSpacing: 1,
        fontWeight: '600',
    },
    content: {
        flex: 1,
    },
    scrollContent: {
        padding: 24,
        paddingBottom: 40,
    },
    sectionTitle: {
        fontSize: 20,
        fontWeight: '700',
        color: '#FFFFFF',
        marginBottom: 20,
        letterSpacing: -0.5,
        marginTop: 10,
    },
    formCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 16,
        padding: 20,
        marginBottom: 30,
        borderWidth: 1,
        borderColor: '#2C2C2C',
    },
    formTitle: {
        fontSize: 18,
        fontWeight: '700',
        color: '#FFF',
        marginBottom: 16,
    },
    input: {
        backgroundColor: '#121212',
        borderRadius: 8,
        borderWidth: 1,
        borderColor: '#333',
        color: '#FFF',
        padding: 12,
        marginBottom: 12,
        fontSize: 16,
    },
    textArea: {
        height: 80,
        textAlignVertical: 'top',
    },
    formActions: {
        flexDirection: 'row',
        justifyContent: 'flex-end',
        gap: 10,
    },
    submitButton: {
        backgroundColor: '#007AFF', // Clean blue for action
        paddingVertical: 12,
        paddingHorizontal: 20,
        borderRadius: 8,
        alignItems: 'center',
    },
    submitButtonText: {
        color: '#FFF',
        fontWeight: '600',
        fontSize: 15,
    },
    cancelButton: {
        paddingVertical: 12,
        paddingHorizontal: 20,
        borderRadius: 8,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#333',
    },
    cancelButtonText: {
        color: '#AAA',
        fontWeight: '600',
        fontSize: 15,
    },
    eventsGrid: {
        marginBottom: 30,
    },
    eventCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 16,
        padding: 24,
        marginBottom: 16,
        borderWidth: 1,
        borderColor: '#2C2C2C',
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
        marginBottom: 8,
    },
    eventName: {
        fontSize: 18,
        fontWeight: '700',
        color: '#FFFFFF',
        flex: 1,
        marginRight: 10,
    },
    eventId: {
        fontSize: 12,
        color: '#666',
        fontFamily: Platform.OS === 'ios' ? 'Courier' : 'monospace',
    },
    eventDescription: {
        fontSize: 15,
        color: '#AAAAAA',
        lineHeight: 22,
        marginBottom: 20,
    },
    cardActions: {
        flexDirection: 'row',
        gap: 12,
    },
    cardAction: {
        paddingVertical: 8,
        paddingHorizontal: 16,
        borderRadius: 8,
        borderWidth: 1,
    },
    editAction: {
        backgroundColor: '#2A2A2A',
        borderColor: '#333',
    },
    deleteAction: {
        backgroundColor: 'rgba(255, 59, 48, 0.1)',
        borderColor: 'rgba(255, 59, 48, 0.3)',
    },
    cardActionText: {
        color: '#FFFFFF',
        fontSize: 13,
        fontWeight: '600',
    },
    deleteText: {
        color: '#FF453A',
    },
    emptyText: {
        color: '#666',
        fontStyle: 'italic',
        textAlign: 'center',
        marginTop: 20,
    },
    backButton: {
        backgroundColor: '#FFFFFF',
        paddingVertical: 18,
        borderRadius: 14,
        alignItems: 'center',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.1,
        shadowRadius: 4,
        marginTop: 10,
    },
    backButtonText: {
        color: '#000000',
        fontSize: 16,
        fontWeight: '700',
        letterSpacing: 0.5,
    },
});
