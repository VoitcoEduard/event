import React, { useState, useEffect, useRef } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, TextInput, Alert, Platform, useWindowDimensions } from 'react-native';
import { StatusBar } from 'expo-status-bar';

// Use environment variable or default to localhost for development
const API_URL = typeof window !== 'undefined' && window.location
    ? (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
        ? 'http://localhost:8080'
        : `http://${window.location.hostname}:8080`)
    : 'http://localhost:8080';

const EventCard = ({ event, onEdit, onDelete, onViewDetails }) => (
    <View style={styles.eventCard}>
        <TouchableOpacity style={styles.cardHeader} onPress={() => onViewDetails(event)}>
            <View style={styles.cardHeaderContent}>
                <Text style={styles.eventName} numberOfLines={1}>{event.name}</Text>
                <Text style={styles.eventId}>ID: {event.id}</Text>
            </View>
            <View style={[styles.statusBadge, event.isActive ? styles.statusActive : styles.statusInactive]}>
                <Text style={styles.statusText}>{event.isActive ? 'ACTIVE' : 'INACTIVE'}</Text>
            </View>
        </TouchableOpacity>
        <Text style={styles.eventDescription} numberOfLines={2}>{event.description}</Text>
        <View style={styles.cardActions}>
            <TouchableOpacity style={[styles.cardAction, styles.viewAction]} onPress={() => onViewDetails(event)}>
                <Text style={styles.cardActionText}>Manage</Text>
            </TouchableOpacity>
            <TouchableOpacity style={[styles.cardAction, styles.editAction]} onPress={() => onEdit(event)}>
                <Text style={styles.cardActionText}>Edit</Text>
            </TouchableOpacity>
            <TouchableOpacity style={[styles.cardAction, styles.deleteAction]} onPress={() => onDelete(event.id)}>
                <Text style={[styles.cardActionText, styles.deleteText]}>Delete</Text>
            </TouchableOpacity>
        </View>
    </View>
);

export default function AdminScreen({ onNavigate, setEvent }) {
    const [events, setEvents] = useState([]);
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const [editingId, setEditingId] = useState(null);
    const [isLoading, setIsLoading] = useState(false);
    const [showForm, setShowForm] = useState(false);
    const scrollViewRef = useRef(null);

    const { width } = useWindowDimensions();
    const isDesktop = width >= 768;

    useEffect(() => {
        fetchEvents();
    }, []);

    const fetchEvents = async () => {
        setIsLoading(true);
        try {
            const response = await fetch(`${API_URL}/events`);
            const data = await response.json();
            if (Array.isArray(data)) {
                setEvents(data);
            } else {
                console.error('Fetched data is not an array:', data);
                setEvents([]);
            }
        } catch (error) {
            console.error('Error fetching events:', error);
            Alert.alert('Error', 'Failed to fetch events');
        } finally {
            setIsLoading(false);
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
                setShowForm(false);
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
        setShowForm(true);
        // Scroll to top to show the form
        scrollViewRef.current?.scrollTo({ x: 0, y: 0, animated: true });
    };

    const handleCreateNew = () => {
        setName('');
        setDescription('');
        setEditingId(null);
        setShowForm(true);
        scrollViewRef.current?.scrollTo({ x: 0, y: 0, animated: true });
    };

    const handleCancelForm = () => {
        setShowForm(false);
        setEditingId(null);
        setName('');
        setDescription('');
    };

    const handleViewDetails = (event) => {
        setEvent(event);
        onNavigate('admin_event_details');
    };


    return (
        <View style={styles.container}>
            <StatusBar style="light" />

            <View style={styles.header}>
                <View>
                    <Text style={styles.headerTitle}>Event Admin</Text>
                    <Text style={styles.headerSubtitle}>Manage Events Listings</Text>
                </View>
                {!showForm && (
                    <TouchableOpacity style={styles.createButton} onPress={handleCreateNew}>
                        <Text style={styles.createButtonText}>+ New Event</Text>
                    </TouchableOpacity>
                )}
            </View>

            <ScrollView
                ref={scrollViewRef}
                style={styles.content}
                contentContainerStyle={styles.scrollContent}
                showsVerticalScrollIndicator={false}
            >
                {showForm && (
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
                            <TouchableOpacity
                                style={styles.cancelButton}
                                onPress={handleCancelForm}
                            >
                                <Text style={styles.cancelButtonText}>Cancel</Text>
                            </TouchableOpacity>
                            <TouchableOpacity style={styles.submitButton} onPress={handleSubmit}>
                                <Text style={styles.submitButtonText}>{editingId ? 'Update Event' : 'Create Event'}</Text>
                            </TouchableOpacity>
                        </View>
                    </View>
                )}

                <View style={styles.sectionHeader}>
                    <Text style={styles.sectionTitle}>
                        {events.length > 0 ? 'All Events' : 'No Events'}
                    </Text>
                </View>

                <View style={[styles.eventsGrid, isDesktop && styles.eventsGridDesktop]}>
                    {isLoading ? (
                        <Text style={styles.emptyText}>Loading events...</Text>
                    ) : (
                        <>
                            {events.map((event) => (
                                <EventCard
                                    key={event.id}
                                    event={event}
                                    onEdit={handleEdit}
                                    onDelete={handleDelete}
                                    onViewDetails={handleViewDetails}
                                />
                            ))}
                            {events.length === 0 && !showForm && (
                                <View style={styles.emptyStateContainer}>
                                    <Text style={styles.emptyText}>No events found.</Text>
                                    <TouchableOpacity style={styles.createLink} onPress={handleCreateNew}>
                                        <Text style={styles.createLinkText}>Create one now</Text>
                                    </TouchableOpacity>
                                </View>
                            )}
                        </>
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
        paddingVertical: 20,
        paddingHorizontal: 24,
        backgroundColor: '#1E1E1E',
        borderBottomWidth: 1,
        borderBottomColor: '#2C2C2C',
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    headerTitle: {
        fontSize: 24,
        fontWeight: '800',
        color: '#FFFFFF',
        letterSpacing: -0.5,
        marginBottom: 4,
    },
    headerSubtitle: {
        fontSize: 13,
        color: '#888888',
        textTransform: 'uppercase',
        letterSpacing: 1,
        fontWeight: '600',
    },
    createButton: {
        backgroundColor: '#007AFF',
        paddingVertical: 10,
        paddingHorizontal: 16,
        borderRadius: 8,
    },
    createButtonText: {
        color: '#FFFFFF',
        fontWeight: '600',
        fontSize: 14,
    },
    content: {
        flex: 1,
    },
    scrollContent: {
        padding: 24,
        paddingBottom: 40,
        alignItems: 'center',
    },
    sectionHeader: {
        width: '100%',
        maxWidth: 1200,
        marginBottom: 16,
        alignSelf: 'center',
    },
    sectionTitle: {
        fontSize: 18,
        fontWeight: '700',
        color: '#FFFFFF',
        letterSpacing: -0.5,
    },
    formCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 12,
        padding: 24,
        marginBottom: 30,
        borderWidth: 1,
        borderColor: '#007AFF', // Highlight form when active
        width: '100%',
        maxWidth: 800,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.3,
        shadowRadius: 8,
        elevation: 5,
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
        fontSize: 15,
    },
    textArea: {
        height: 80,
        textAlignVertical: 'top',
    },
    formActions: {
        flexDirection: 'row',
        justifyContent: 'flex-end',
        gap: 10,
        marginTop: 8,
    },
    submitButton: {
        backgroundColor: '#007AFF',
        paddingVertical: 10,
        paddingHorizontal: 20,
        borderRadius: 8,
        alignItems: 'center',
    },
    submitButtonText: {
        color: '#FFF',
        fontWeight: '600',
        fontSize: 14,
    },
    cancelButton: {
        paddingVertical: 10,
        paddingHorizontal: 20,
        borderRadius: 8,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#333',
    },
    cancelButtonText: {
        color: '#AAA',
        fontWeight: '600',
        fontSize: 14,
    },
    eventsGrid: {
        marginBottom: 30,
        flexDirection: 'column', // Mobile first
        gap: 12,
        width: '100%',
        maxWidth: 1200,
        alignSelf: 'center',
    },
    eventsGridDesktop: {
        flexDirection: 'row',
        flexWrap: 'wrap',
    },
    eventCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 12,
        padding: 16, // Reduced padding
        marginBottom: 0,
        borderWidth: 1,
        borderColor: '#2C2C2C',
        minWidth: 280, // Slightly smaller base width
        flex: 1,
        maxWidth: 400, // Reduced max width to keep cards compact
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
        marginBottom: 8,
    },
    cardHeaderContent: {
        flex: 1,
        marginRight: 8,
    },
    eventName: {
        fontSize: 16, // Smaller title
        fontWeight: '700',
        color: '#FFFFFF',
        marginBottom: 2,
    },
    eventId: {
        fontSize: 11,
        color: '#666',
        fontFamily: Platform.OS === 'ios' ? 'Courier' : 'monospace',
    },
    statusBadge: {
        paddingVertical: 2,
        paddingHorizontal: 6,
        borderRadius: 4,
    },
    statusActive: {
        backgroundColor: 'rgba(48, 209, 88, 0.15)',
    },
    statusInactive: {
        backgroundColor: 'rgba(255, 69, 58, 0.15)',
    },
    statusText: {
        fontSize: 9,
        fontWeight: '700',
        color: '#FFF',
        textTransform: 'uppercase',
    },
    eventDescription: {
        fontSize: 13,
        color: '#AAAAAA',
        lineHeight: 18,
        marginBottom: 16,
        height: 36, // Fixed height for 2 lines approx
    },
    cardActions: {
        flexDirection: 'row',
        gap: 8,
        flexWrap: 'wrap',
    },
    cardAction: {
        paddingVertical: 6,
        paddingHorizontal: 12,
        borderRadius: 6,
        borderWidth: 1,
    },
    viewAction: {
        backgroundColor: 'rgba(255, 255, 255, 0.05)',
        borderColor: '#444',
    },
    editAction: {
        backgroundColor: 'rgba(255, 255, 255, 0.05)',
        borderColor: '#444',
    },
    deleteAction: {
        backgroundColor: 'rgba(255, 59, 48, 0.05)',
        borderColor: 'rgba(255, 59, 48, 0.2)',
    },
    cardActionText: {
        color: '#E0E0E0',
        fontSize: 12,
        fontWeight: '600',
    },
    deleteText: {
        color: '#FF6B60',
    },
    emptyStateContainer: {
        alignItems: 'center',
        padding: 40,
        width: '100%',
    },
    emptyText: {
        color: '#666',
        fontStyle: 'italic',
        textAlign: 'center',
        marginBottom: 10,
    },
    createLink: {
        padding: 10,
    },
    createLinkText: {
        color: '#007AFF',
        fontWeight: '600',
    },
    backButton: {
        backgroundColor: '#2C2C2C', // Darker back button
        paddingVertical: 14,
        paddingHorizontal: 24,
        borderRadius: 10,
        alignItems: 'center',
        alignSelf: 'center',
        marginTop: 10,
        borderWidth: 1,
        borderColor: '#333',
    },
    backButtonText: {
        color: '#FFFFFF',
        fontSize: 14,
        fontWeight: '600',
    },
});
